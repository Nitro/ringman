package ringman

// This file is licensed under the MIT license and derives from
// http://github.com/Nitro/sidecar

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_LoggingBridge(t *testing.T) {
	Convey("LoggingBridge", t, func() {
		bridge := LoggingBridge{testing: true}

		Convey("Properly splits apart and re-levels log messages", func() {
			bridge.Write([]byte("2016/06/24 11:45:33 [DEBUG] memberlist: TCP connection from=172.16.106.1:59598"))

			So(string(bridge.lastLevel), ShouldEqual, "[DEBUG]")
			So(string(bridge.lastMessage), ShouldEqual, "memberlist: TCP connection from=172.16.106.1:59598")

			bridge.Write([]byte("2016/06/24 11:45:33 [WARN] memberlist: Something something"))

			So(string(bridge.lastLevel), ShouldEqual, "[WARN]")
			So(string(bridge.lastMessage), ShouldEqual, "memberlist: Something something")
		})

		Convey("Handles writes that have more than one message", func() {
			bridge.Write(
				[]byte("2016/06/24 11:45:33 [DEBUG] memberlist: TCP connection from=172.16.106.1:59598\nasdf"),
			)

			So(string(bridge.lastMessage), ShouldEqual, "memberlist: TCP connection from=172.16.106.1:59598")
		})
	})
}
