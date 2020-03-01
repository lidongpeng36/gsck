package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lidongpeng36/gsck/formatter"
	"github.com/lidongpeng36/gsck/util"
	"golang.org/x/crypto/ssh"
)

var randSrc = rand.NewSource(time.Now().UnixNano())
var randGen = rand.New(randSrc)

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
	alias    string
	port     string
	cmd      string
	timeout  int64
	client   *ssh.Client
	session  *ssh.Session
	config   *ssh.ClientConfig
	retry    int
	transfer *TransferFile
}

func (sc sshClient) exec() (stdout, stderr string, rc int, err error) {
	timeout := sc.timeout
	for retry := sc.retry; retry >= 0; retry-- {
		connectError := make(chan error, 0)
		go func() {
			var _err error
			sc.client, _err = ssh.Dial("tcp", sc.hostname+":"+sc.port, sc.config)
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
			ms := randGen.Int63n(1000)
			if sc.client != nil {
				_ = sc.client.Close()
			}
			time.Sleep(time.Duration(ms) * time.Millisecond)
		} else {
			rc = -1
			return
		}
	}
	defer func() {
		// Close client
		_ = sc.client.Close()
	}()
	sc.session, err = sc.client.NewSession()
	if err != nil {
		return
	}
	defer func() {
		// Close Session
		_ = sc.session.Close()
	}()
	var stdoutBuf, stderrBuf bytes.Buffer
	sc.session.Stdout = &stdoutBuf
	sc.session.Stderr = &stderrBuf
	execError := make(chan error, 0)
	if sc.transfer != nil {
		go func() {
			stdin, _err := sc.session.StdinPipe()
			defer func() { _ = stdin.Close() }()
			if _err != nil {
				execError <- _err
			}
			fmt.Fprintf(stdin, "C%v %v %v\n", sc.transfer.Perm, len(sc.transfer.Data), sc.transfer.Basename)
			_, _ = stdin.Write(sc.transfer.Data)
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

func (sc *sshClient) output() *formatter.Output {
	stdout, stderr, rc, clientErr := sc.exec()
	output := &formatter.Output{
		Hostname: sc.hostname,
		Alias:    sc.alias,
		ExitCode: rc,
	}
	if clientErr == nil {
		output.Stdout = stdout
		output.Stderr = stderr
	} else {
		output.Error = clientErr.Error()
	}
	return output
}

type sshExecutor struct {
	config  *ssh.ClientConfig
	clients []*sshClient
	data    *Parameter
}

func (ss *sshExecutor) assembleSSHCmd(cmd string) string {
	var transferCmd string
	if ss.data.NeedTransferFile() {
		trans := ss.data.Transfer
		transferCmd = "cd " + trans.Destination +
			" && /usr/bin/scp -qrt ." + " && " +
			fmt.Sprintf("echo '%s saved.'", trans.Dst)
	}
	transferCmd = ss.data.WrapCmdWithHook(transferCmd)
	return util.WrapCmdBefore(cmd, transferCmd)
}

// pragma mark - Worker Interface

func (ss *sshExecutor) Name() string {
	return "ssh"
}

func (ss *sshExecutor) Init(data *Parameter) error {
	ss.data = data
	hostinfoList := data.HostInfoList
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
	if data.Passwd != "" {
		authMethod = append(authMethod, ssh.Password(data.Passwd))
	}
	ss.config = &ssh.ClientConfig{
		User: data.User,
		Auth: authMethod,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	ss.clients = make([]*sshClient, len(hostinfoList))
	var transfer *TransferFile
	if data.NeedTransferFile() {
		transfer = data.Transfer
	}
	retry := data.Retry
	for i, info := range hostinfoList {
		hostname := info.Host
		cmdFinal := ss.assembleSSHCmd(info.Cmd)
		client := &sshClient{
			hostname: hostname,
			alias:    info.Alias,
			port:     info.Port,
			config: &ssh.ClientConfig{
				User: info.User,
				Auth: authMethod,
				HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
					return nil
				},
			},
			cmd:      cmdFinal,
			retry:    retry,
			transfer: transfer,
			timeout:  data.Timeout,
		}
		ss.clients[i] = client
	}
	return nil
}

func (ss *sshExecutor) Execute(done <-chan struct{}) (<-chan *formatter.Output, <-chan error) {
	ch := make(chan *formatter.Output)
	errc := make(chan error)
	go func() {
		var wg sync.WaitGroup
		sem := make(chan bool, ss.data.Concurrency)
		for _, c := range ss.clients {
			wg.Add(1)
			go func(client *sshClient) {
				sem <- true
				defer func() {
					wg.Done()
					<-sem
				}()
				select {
				case ch <- client.output():
				case <-done:
				}
			}(c)
		}
		go func() {
			wg.Wait()
			close(errc)
			close(ch)
		}()
	}()
	return ch, errc

}
