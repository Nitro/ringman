package ringman

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/serialx/hashring"
)

var (
	ErrNilManager error = errors.New("HashRingManager has not been initialized!")
)

const (
	CmdAddNode    = iota
	CmdRemoveNode = iota
	CmdGetNode    = iota
)

type HashRingManager struct {
	HashRing *hashring.HashRing
	cmdChan  chan RingCommand
	started  bool
}

type RingCommand struct {
	Command   int
	NodeName  string
	Key       string
	ReplyChan chan *RingReply
}

type RingReply struct {
	Error error
	Nodes []string
}

func NewHashRingManager(nodeList []string) *HashRingManager {
	return &HashRingManager{
		HashRing: hashring.New(nodeList),
		cmdChan:  make(chan RingCommand),
	}
}

// Run runs in a loop over the contents of cmdChan and processes the
// incoming work. This acts as the synchronization around the HashRing
// itself which is not mutable and has to be replaced on each command.
func (r *HashRingManager) Run() error {
	if r == nil {
		return ErrNilManager
	}

	r.started = true

	// The cmdChan is used to synchronize all the access to the HashRing
	for msg := range r.cmdChan {
		switch msg.Command {
		case CmdAddNode:
			r.HashRing = r.HashRing.AddNode(msg.NodeName)
		case CmdRemoveNode:
			r.HashRing = r.HashRing.RemoveNode(msg.NodeName)
		case CmdGetNode:
			node, ok := r.HashRing.GetNode(msg.Key)
			var err error
			if !ok {
				err = errors.New("No nodes in ring!")
			}
			msg.ReplyChan <- &RingReply{Error: err, Nodes: []string{node}}
		default:
			log.Errorf("Received unexpected command %d", msg.Command)
		}
	}

	return nil
}

// Stop the HashRingManager from running. This is currently permanent since
// the internal cmdChan it closes can't be re-opened.
func (r *HashRingManager) Stop() {
	if r.cmdChan != nil {
		close(r.cmdChan)
		r.cmdChan = nil // Prevent issues reading on closed channel
	}
	r.started = false
}

// wrapCommand handles validation of dependencies for the various commands.
func (r *HashRingManager) wrapCommand(fn func() error) error {
	if r == nil {
		return ErrNilManager
	}
	if !r.started {
		return errors.New("HashRingManager has not been started")
	}
	if r.cmdChan == nil {
		return errors.New("HashRingManager has a nil command channel. May not be initialized!")
	}

	return fn()
}

// AddNode is a blocking call that will send an add message on the message
// channel for the HashManager.
func (r *HashRingManager) AddNode(nodeName string) error {
	return r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdAddNode, nodeName, "", nil}
		return nil
	})
}

// RemoveNode is a blocking call that will send an add message on the message
// channel for the HashManager.
func (r *HashRingManager) RemoveNode(nodeName string) error {
	return r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdRemoveNode, nodeName, "", nil}
		return nil
	})
}

// GetNodes requests the current list of nodes from the ring.
func (r *HashRingManager) GetNode(key string) (string, error) {
	replyChan := make(chan *RingReply)

	err := r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdGetNode, "", key, replyChan}
		return nil
	})

	if err != nil {
		return "", nil
	}

	reply := <-replyChan
	return reply.Nodes[0], reply.Error
}
