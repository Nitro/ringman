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
	CmdAddNode        = iota
	CmdRemoveNode     = iota
	CmdGetNode        = iota
)

const (
	CommandChannelLength = 10 // How big a buffer on our mailbox channel?
)

type HashRingManager struct {
	HashRing    *hashring.HashRing
	cmdChan     chan RingCommand
	started     bool
	waitChan    chan struct{}
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

// NewHashRingManager returns a properly configured HashRingManager. It accepts
// zero or mode nodes to initialize the ring with.
func NewHashRingManager(nodeList []string) *HashRingManager {
	return &HashRingManager{
		HashRing: hashring.New(nodeList),
		cmdChan:  make(chan RingCommand, CommandChannelLength),
		waitChan: make(chan struct{}),
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
	if r.waitChan != nil {
		close(r.waitChan)
	}
	r.waitChan = nil

	// The cmdChan is used to synchronize all the access to the HashRing
	for msg := range r.cmdChan {
		switch msg.Command {
		case CmdAddNode:
			log.Debugf("Adding node %s", msg.NodeName)
			r.HashRing = r.HashRing.AddNode(msg.NodeName)

		case CmdRemoveNode:
			log.Debugf("Removing node %s", msg.NodeName)
			r.HashRing = r.HashRing.RemoveNode(msg.NodeName)

		case CmdGetNode:
			node, ok := r.HashRing.GetNode(msg.Key)
			var err error
			if !ok {
				err = errors.New("No nodes in ring!")
			}

			msg.ReplyChan <- &RingReply{
				Error: err,
				Nodes: []string{node},
			}

		default:
			log.Errorf("Received unexpected command %d", msg.Command)
		}
	}
	log.Warnf("Closed cmdChan")

	return nil
}

// Pending returns the number of pending commands in the command channel
func (r *HashRingManager) Pending() int {
	return len(r.cmdChan)
}

// Wait for the HashRingManager to be running. Used to synchronize with a
// goroutine running in the background.
func (r *HashRingManager) Wait() {
	<-r.waitChan
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
	err := r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdAddNode, nodeName, "", nil}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// RemoveNode is a blocking call that will send an add message on the message
// channel for the HashManager.
func (r *HashRingManager) RemoveNode(nodeName string) error {
	err := r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdRemoveNode, nodeName, "", nil}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// GetNode requests a node from the ring to serve the provided key
func (r *HashRingManager) GetNode(key string) (string, error) {
	replyChan := make(chan *RingReply)
	err := r.wrapCommand(func() error {
		r.cmdChan <- RingCommand{CmdGetNode, "", key, replyChan}
		return nil
	})

	if err != nil {
		return "", err
	}

	reply := <-replyChan
	close(replyChan)
	replyChan = nil

	return reply.Nodes[0], reply.Error
}
