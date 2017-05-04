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
			ringMgr.Wait()

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
		ringMgr := NewHashRingManager([]string{"kjartan"})
		go ringMgr.Run()
		ringMgr.Wait()

		Convey("AddNode adds a node which is returned from GetNode", func() {

			err := ringMgr.AddNode("njal")
			So(err, ShouldBeNil)

			node, err := ringMgr.GetNode("foo")
			So(err, ShouldBeNil)
			So(node.NodeName, ShouldEqual, "njal")
		})

		Convey("RemoveNode removes a node", func() {
			ringMgr.RemoveNode("kjartan")

			node, err := ringMgr.GetNode("foo")
			So(err, ShouldNotBeNil)
			So(node.Metadata, ShouldEqual, nil)
			So(node.NodeName, ShouldEqual, "")
		})

		Convey("UpdateMetadata updates the internal map", func() {
			ringMgr.UpdateMetadataSync("kjartan", &RingMetadata{Port: "1234"})

			So(ringMgr.Metadata["kjartan"].Port, ShouldEqual, "1234")
		})

		Convey("With error conditions", func() {
			Convey("does not blow up on nil receiver", func() {
				var broken *HashRingManager

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
			})

			Convey("does not try to run if not started", func() {
				broken := &HashRingManager{started: false}
				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
			})

			Convey("does not blow up on nil a closed channel", func() {
				var broken HashRingManager
				broken.started = true

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
			})
		})
	})
}
