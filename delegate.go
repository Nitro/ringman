package ringman

import (
	"github.com/Nitro/memberlist"
	log "github.com/Sirupsen/logrus"
)

type Delegate struct {
	ringMan *HashRingManager
}

func NewDelegate(ringMan *HashRingManager) *Delegate {
	delegate := Delegate{
		ringMan: ringMan,
	}

	return &delegate
}

func (d *Delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *Delegate) NotifyMsg(message []byte) {
	log.Debugf("NotifyMsg(): %s", string(message))
}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	//log.Debugf("GetBroadcasts(): %d %d", overhead, limit)
	return [][]byte{}
}

func (d *Delegate) LocalState(join bool) []byte {
	log.Debugf("LocalState(): %b", join)
	return []byte{}
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
	log.Debugf("MergeRemoteState(): %s %t", string(buf), join)
}

func (d *Delegate) NotifyJoin(node *memberlist.Node) {
	log.Debugf("NotifyJoin(): %s %s", node.Name, string(node.Meta))
	d.ringMan.AddNode(node.Name)
}

func (d *Delegate) NotifyLeave(node *memberlist.Node) {
	log.Debugf("NotifyLeave(): %s", node.Name)
	d.ringMan.RemoveNode(node.Name)
}

func (d *Delegate) NotifyUpdate(node *memberlist.Node) {
	log.Debugf("NotifyUpdate(): %s", node.Name)
}
