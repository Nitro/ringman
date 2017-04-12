package ringman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_NewHashRingManager(t *testing.T) {
	Convey("NewHashRingManager()", t, func() {
		hostList := []string{"njal", "kjartan"}
		ringMgr := NewHashRingManager(hostList)

		Convey("returns a properly configured HashRingManager", func() {
			So(ringMgr.cmdChan, ShouldNotBeNil)
			So(ringMgr.HashRing, ShouldNotBeNil)
			So(ringMgr.started, ShouldBeFalse)
		})
	})
}

func Test_Run(t *testing.T) {
	Convey("Run()", t, func() {
		hostList := []string{"njal", "kjartan"}

		ringMgr := NewHashRingManager(hostList)

		Convey("sets 'started' to true", func() {
			go ringMgr.Run()
			ringMgr.cmdChan <-RingCommand{}
			So(ringMgr.started, ShouldBeTrue)
			ringMgr.Stop()
		})

		Convey("doesn't blow up on a nil receiver", func() {
			var broken *HashRingManager
			So(func() { broken.Run() }, ShouldNotPanic)
		})
	})
}

func Test_Commands(t *testing.T) {
	Convey("Running commands", t, func() {

		Convey("With error conditions", func() {
			Convey("does not blow up on nil receiver", func() {
				var broken *HashRingManager

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
				So(func() { broken.GetNode("junk") }, ShouldNotPanic)
			})

			Convey("does not try to run if not started", func() {
				broken := &HashRingManager{started: false}
				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
				So(func() { broken.GetNode("junk") }, ShouldNotPanic)
			})

			Convey("does not blow up on nil a closed channel", func() {
				var broken HashRingManager
				broken.started = true

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
				So(func() { broken.GetNode("junk") }, ShouldNotPanic)
			})
		})
	})
}
