package ringman

import (
	"testing"

	director "github.com/relistan/go-director"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_NewHashRingManager(t *testing.T) {
	Convey("NewHashRingManager()", t, func() {
		hostList := []string{"njal", "kjartan"}
		ringMgr := NewHashRingManager(hostList)

		Convey("returns a properly configured HashRingManager", func() {
			So(ringMgr.cmdChan, ShouldNotBeNil)
			So(ringMgr.HashRing, ShouldNotBeNil)
		})
	})
}

func Test_Run(t *testing.T) {
	Convey("Run()", t, func() {
		hostList := []string{"njal", "kjartan"}
		ringMgr := NewHashRingManager(hostList)
		go ringMgr.Run(director.NewFreeLooper(director.ONCE, nil))

		Convey("ringMgr.Ping() returns true when running", func() {

			So(ringMgr.Ping(), ShouldBeTrue)
		})

		Convey("ringMgr can stop", func() {
			So(ringMgr.Ping(), ShouldBeTrue)

			ringMgr.Stop()
			So(ringMgr.cmdChan, ShouldBeNil)

			So(ringMgr.Ping(), ShouldBeFalse)
		})

		Convey("doesn't blow up on a nil receiver", func() {
			var broken *HashRingManager
			So(func() { broken.Run(nil) }, ShouldNotPanic)
		})
	})
}

func Test_Commands(t *testing.T) {
	Convey("Running commands", t, func() {
		ringMgr := NewHashRingManager([]string{"kjartan"})

		Convey("AddNode adds a node which is returned from GetNode", func() {
			go ringMgr.Run(director.NewFreeLooper(3, nil))
			// Make sure the RingManager is started
			So(ringMgr.Ping(), ShouldBeTrue)

			err := ringMgr.AddNode("njal")
			So(err, ShouldBeNil)

			node, err := ringMgr.GetNode("foo")
			So(err, ShouldBeNil)
			So(node, ShouldEqual, "njal")
		})

		Convey("RemoveNode removes a node", func() {
			go ringMgr.Run(director.NewFreeLooper(3, nil))
			// Make sure the RingManager is started
			So(ringMgr.Ping(), ShouldBeTrue)

			ringMgr.RemoveNode("kjartan")

			node, err := ringMgr.GetNode("foo")
			So(err, ShouldNotBeNil)
			So(node, ShouldEqual, "")
		})

		Convey("Ping responds as up, in a timely manner", func() {
			go ringMgr.Run(director.NewFreeLooper(director.ONCE, nil))

			result := ringMgr.Ping()

			So(result, ShouldBeTrue)
		})

		Convey("Ping fails when the manager is not running", func() {
			go ringMgr.Run(director.NewFreeLooper(director.ONCE, nil))
			// Make sure the RingManager is started
			So(ringMgr.Ping(), ShouldBeTrue)

			ringMgr.Stop()

			So(ringMgr.Ping(), ShouldBeFalse)
		})

		Convey("With error conditions", func() {
			Convey("does not blow up on nil receiver", func() {
				var broken *HashRingManager

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
			})

			Convey("does not try to run if not started", func() {
				broken := &HashRingManager{}

				So(func() { broken.AddNode("junk") }, ShouldNotPanic)
				So(func() { broken.RemoveNode("junk") }, ShouldNotPanic)
			})
		})
	})
}

func Test_Ring(t *testing.T) {
	Convey("SidecarRing and MemberlistRing implement Ring", t, func() {
		var ring Ring

		So(func() { ring = &MemberlistRing{} }, ShouldNotPanic)
		So(func() { ring = &SidecarRing{} }, ShouldNotPanic)
	})

}
