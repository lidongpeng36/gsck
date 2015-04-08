package p2p

import ()

var _p2pmgr *Mgr

func init() {
	_p2pmgr = new(Mgr)
}

// Constructor is constructor for all P2P implementations.
type Constructor func() P2P

var constructorMap = map[string]Constructor{}

// RegisterP2P Register a new P2P for Mgr to choose
func RegisterP2P(builder Constructor) {
	constructorMap[builder().Name()] = builder
}

// P2P typically use a torrent file to share files
type P2P interface {
	Name() string
	Mkseed() error
	// Set source and destination dir
	SetTransfer(src, dst string)
	// If a torrent file needs to be transferred
	NeedTransferFile() bool
	// Torrent file path
	TransferFilePath() string
	ClientCmd() string
	Available() bool
}

// Mgr is managers concreate P2P tools (P2P interface)
type Mgr struct {
	saveDir string
	source  string
	P2P
}

// GetMgr returns singleton instance of Mgr
func GetMgr() *Mgr {
	return _p2pmgr
}

// Available returns if we can find any available P2P implementation
func Available() bool {
	if _p2pmgr.P2P != nil {
		return true
	}
	for _, builder := range constructorMap {
		p := builder()
		if p.Available() {
			_p2pmgr.P2P = p
			return true
		}
	}
	return false
}

// SetTransfer sets source and destination for Copy.
func (mgr *Mgr) SetTransfer(src, dst string) {
	mgr.source = src
	mgr.saveDir = dst
	mgr.P2P.SetTransfer(src, dst)
}

// Source gives source path (local)
func (mgr *Mgr) Source() string {
	return mgr.source
}

// DstDir destination dir path (remote)
func (mgr *Mgr) DstDir() string {
	return mgr.saveDir
}
