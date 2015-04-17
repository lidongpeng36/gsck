package executor

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"errors"
	"fmt"
	"github.com/EvanLi/gsck/config"
	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/util"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var randSrc = rand.New(rand.NewSource(1000))

func init() {
	RegisterWorker(func() Worker {
		return &sshExecutor{
			config: nil,
		}
	})
}

func getKeyFile(t string) (key ssh.Signer, err error) {
	file := path.Join(os.Getenv("HOME"), ".ssh", "id_"+t)
	fi, err := os.Lstat(file)
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		file, err = filepath.EvalSymlinks(file)
		if err != nil {
			return
		}
	}
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return
	}
	return
}

type sshClient struct {
	hostname string
	cmd      string
	client   *ssh.Client
	session  *ssh.Session
	config   *ssh.ClientConfig
	retry    int
	transfer *TransferFile
}

func (sc sshClient) exec() (stdout, stderr string, rc int, err error) {
	timeout := _executor.Timeout
	for retry := sc.retry; retry >= 0; retry-- {
		connectError := make(chan error, 1)
		go func() {
			var _err error
			sc.client, _err = ssh.Dial("tcp", sc.hostname+":22", sc.config)
			connectError <- _err
		}()
		go func() {
			if timeout > 0 {
				time.Sleep(time.Duration(timeout) * time.Second)
				connectError <- errors.New("Connection Timeout.")
			}
		}()
		if err = <-connectError; err == nil {
			break
		} else if retry > 0 {
			ms := randSrc.Int63n(1000)
			if sc.client != nil {
				sc.client.Close()
			}
			time.Sleep(time.Duration(ms) * time.Millisecond)
		} else {
			rc = -1
			return
		}
	}
	defer func() {
		// Close client
		sc.client.Close()
	}()
	sc.session, err = sc.client.NewSession()
	if err != nil {
		return
	}
	defer func() {
		// Close Session
		sc.session.Close()
	}()
	var stdoutBuf, stderrBuf bytes.Buffer
	sc.session.Stdout = &stdoutBuf
	sc.session.Stderr = &stderrBuf
	execError := make(chan error, 1)
	if sc.transfer != nil {
		go func() {
			stdin, _err := sc.session.StdinPipe()
			defer stdin.Close()
			if _err != nil {
				execError <- _err
			}
			fmt.Fprintf(stdin, "C%v %v %v\n", sc.transfer.Perm, len(sc.transfer.Data), sc.transfer.Basename)
			stdin.Write(sc.transfer.Data)
			fmt.Fprint(stdin, "\x00")
		}()
	}
	go func() {
		_err := sc.session.Run(sc.cmd)
		stdout = strings.TrimSpace(stdoutBuf.String())
		stderr = strings.TrimSpace(stderrBuf.String())
		if _err != nil {
			sshErr := _err.(*ssh.ExitError)
			rc = sshErr.ExitStatus()
			execError <- errors.New(stderr)
		}
		execError <- nil
	}()
	go func() {
		if timeout > 0 {
			time.Sleep(time.Duration(timeout) * time.Second)
			execError <- errors.New("Execution Timeout.")
		}
	}()
	err = <-execError
	return
}

type sshExecutor struct {
	config  *ssh.ClientConfig
	clients []sshClient
}

func assembleSSHCmd(cmd string) string {
	var transferCmd string
	if _executor.NeedTransferFile() {
		trans := _executor.Transfer
		transferCmd = "cd " + trans.Destination +
			" && /usr/bin/scp -qrt ." + " && " +
			fmt.Sprintf("echo '%s saved.'", trans.Dst)
	}
	transferCmd = _executor.WrapCmdWithHook(transferCmd)
	return util.WrapCmdBefore(cmd, transferCmd)
}

// pragma mark - Worker Interface

func (ss *sshExecutor) Name() string {
	return "ssh"
}

func (ss *sshExecutor) Init() error {
	hostlist := _executor.Hostlist
	cmd := _executor.Cmd
	authKeys := make([]ssh.Signer, 0, 2)
	for _, t := range []string{"rsa", "dsa"} {
		key, err := getKeyFile(t)
		if err != nil {
			continue
		}
		authKeys = append(authKeys, key)
	}
	authMethod := []ssh.AuthMethod{
		ssh.PublicKeys(authKeys...),
	}
	if _executor.Passwd != "" {
		authMethod = append(authMethod, ssh.Password(_executor.Passwd))
	}
	ss.config = &ssh.ClientConfig{
		User: _executor.User,
		Auth: authMethod,
	}
	ss.clients = make([]sshClient, 0, len(hostlist))
	var transfer *TransferFile
	if _executor.NeedTransferFile() {
		transfer = _executor.Transfer
	}
	cmdFinal := assembleSSHCmd(cmd)
	retry := config.GetInt("retry")
	for _, hostname := range hostlist {
		client := sshClient{
			hostname: hostname,
			config:   ss.config,
			cmd:      cmdFinal,
			retry:    retry,
			transfer: transfer,
		}
		ss.clients = append(ss.clients, client)
	}
	return nil
}

func (ss *sshExecutor) Execute() (err error) {
	count := int64(len(ss.clients))
	concurrency := _executor.Concurrency
	groups := count/concurrency + 1
	var i int64
	for i = 0; i < groups; i++ {
		start := concurrency * i
		end := start + concurrency
		if end > count {
			end = count
		}
		if start == end {
			break
		}
		list := ss.clients[start:end]
		var wg sync.WaitGroup
		for _, c := range list {
			wg.Add(1)
			go func(client sshClient) {
				stdout, stderr, rc, clientErr := client.exec()
				output := &formatter.Output{
					Hostname: client.hostname,
					ExitCode: rc,
				}
				if clientErr == nil {
					output.Stdout = stdout
					output.Stderr = stderr
				} else {
					output.Error = clientErr.Error()
				}
				_executor.AddOutput(output)
				defer wg.Done()
			}(c)
		}
		wg.Wait()
	}
	return
}
