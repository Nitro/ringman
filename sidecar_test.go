package ringman

import (
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_NewSidecarRing(t *testing.T) {
	Convey("NewSidecarRing()", t, func() {
		ring := NewSidecarRing("127.0.0.1", "some-svc", 9999)

		Convey("Returns a properly configured struct", func() {
			So(ring.Manager, ShouldNotBeNil)
			So(ring.managerLooper, ShouldNotBeNil)
			So(ring.svcName, ShouldEqual, "some-svc")
			So(ring.svcPort, ShouldEqual, 9999)
		})

		Reset(func() {
			ring.Shutdown()
		})
	})
}

func Test_HttpMux(t *testing.T) {
	Convey("HttpMux()", t, func() {
		ring := NewSidecarRing("127.0.0.1", "some-svc", 9999)

		Convey("Returns a valid Mux", func() {
			So(ring.HttpMux(), ShouldNotBeNil)
		})
	})
}

func Test_onUpdate(t *testing.T) {
	Convey("onUpdate()", t, func() {
		svcName := "some-svc"

		ring := NewSidecarRing("127.0.0.1", svcName, 9999)
		state := catalog.NewServicesState()

		svc := service.Service{
			ID:       "deadbeef123",
			Name:     svcName,
			Image:    "101deadbeef",
			Hostname: "some-host",
			Status:   service.ALIVE,
			Ports:    []service.Port{{Port: 23423, ServicePort: 9999, IP: "127.0.0.1"}},
		}

		state.AddServiceEntry(svc)

		Convey("adds new nodes to the ring", func() {
			So(len(ring.nodes), ShouldEqual, 0)
			ring.onUpdate(state)
			So(len(ring.nodes), ShouldEqual, 1)

			node, err := ring.Manager.GetNode("anything")
			So(err, ShouldBeNil)
			So(node, ShouldEqual, "127.0.0.1:23423")
		})

		Convey("removes old nodes to the ring", func() {
			svc2 := service.Service{
				ID:       "abbaabbaabba",
				Name:     svcName,
				Image:    "101deadbeef",
				Hostname: "some-host",
				Status:   service.ALIVE,
				Ports:    []service.Port{{Port: 12345, ServicePort: 9999}},
			}
			state.AddServiceEntry(svc2)

			ring.onUpdate(state)
			So(len(ring.nodes), ShouldEqual, 2)
			ring.onUpdate(catalog.NewServicesState())
			So(len(ring.nodes), ShouldEqual, 0)

			node, err := ring.Manager.GetNode("anything")
			So(err.Error(), ShouldContainSubstring, "No nodes in ring")
			So(node, ShouldEqual, "")
		})

		Reset(func() {
			ring.Shutdown()
		})
	})
}

func Test_SidecarHttpGetNodeHandler(t *testing.T) {
	Convey("SidecarHttpGetNodeHandler()", t, func() {
		ring := NewSidecarRing("127.0.0.1:7777", "some-svc", 31337)

		req := httptest.NewRequest("GET", "/services/boccacio.json", nil)
		recorder := httptest.NewRecorder()

		state := catalog.NewServicesState()

		svc := service.Service{
			ID:       "deadbeef123",
			Name:     "some-svc",
			Image:    "101deadbeef",
			Hostname: "some-host",
			Status:   service.ALIVE,
			Ports:    []service.Port{{Port: 23423, ServicePort: 31337, IP: "127.0.0.1"}},
		}

		state.AddServiceEntry(svc)

		ring.onUpdate(state)

		Convey("returns a 404 when no key is provided", func() {
			ring.HttpGetNodeHandler(recorder, req)

			So(recorder.Result().StatusCode, ShouldEqual, 404)
		})

		Convey("returns a node when a key is provided", func() {
			form := url.Values{}
			form.Set("key", "bocaccio")
			req.Form = form

			ring.HttpGetNodeHandler(recorder, req)

			bodyBytes, _ := ioutil.ReadAll(recorder.Result().Body)
			body := string(bodyBytes)

			So(recorder.Result().StatusCode, ShouldEqual, 200)
			So(body, ShouldContainSubstring, `"Key": "bocaccio"`)
			So(body, ShouldContainSubstring, `"Node": "127.0.0.1:23423"`)
		})

		Reset(func() {
			ring.Shutdown()
		})
	})
}