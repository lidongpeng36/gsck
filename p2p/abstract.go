package p2p

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lidongpeng36/gsck/util"
	// "path"
	"strings"
)

func init() {
	RegisterP2P(func() P2P {
		return new(AbstractP2P)
	})
}

// AbstractP2P : assemble all server and client commands from config
type AbstractP2P struct {
	src    string
	dst    string
	client string
	mkseed string
}

// pragma mark - P2P interface

// Name is AbstractP2P
func (ap *AbstractP2P) Name() string {
	return "AbstractP2P"
}

// Available tests if p2p config exists
func (ap *AbstractP2P) Available() bool {
	// ap.client = config.GetString("p2p.client")
	// ap.mkseed = config.GetString("p2p.mkseed")
	if ap.client == "" || ap.mkseed == "" {
		return false
	}
	return true
}

// SetTransfer sets src (local) and dst (remote) path
func (ap *AbstractP2P) SetTransfer(src, dst string) {
	ap.src = src
	ap.dst = dst
}

// TransferFilePath returns local.tmpdir/path_to_src.torrent
func (ap *AbstractP2P) TransferFilePath() string {
	// localTmp := config.GetString("local.tmpdir")
	localTmp := "/tmp"
	seperator := string(os.PathSeparator)
	transSrcPath := strings.Replace(ap.src, seperator, "_", -1)
	return localTmp + seperator + transSrcPath + ".torrent"
}

// Mkseed invokes `p2p.mkseed local.tmpdir/path_to_src.torrent`
func (ap *AbstractP2P) Mkseed() (err error) {
	mkseedArray := util.SplitBySpace(ap.mkseed + " " + ap.TransferFilePath())
	args := mkseedArray[1:]
	_, err = exec.Command(mkseedArray[0], args...).Output()
	if err != nil {
		return
	}
	return
}

// NeedTransferFile always return true as normal p2p download needs a torrent file in client.
func (ap *AbstractP2P) NeedTransferFile() bool {
	return true
}

// ClientCmd is `p2p.client remote.tmpdir/path_to_src.torrent`
func (ap *AbstractP2P) ClientCmd() string {
	// torrentPath := path.Join(config.GetString("remote.tmpdir"), ap.TransferFilePath())
	torrentPath := ""
	cmd := fmt.Sprintf("%s %s", ap.client, torrentPath)
	return cmd
}
