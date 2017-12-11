Ringman
=======

This is a consistent hash ring implementation backed by either [our fork of
Hashicorp's Memberlist library](https://github.com/Nitro/memberlist), or
[Sidecar service discovery platform](https://github.com/Nitro/sidecar), and the
[hashring](https://github.com/serialx/hashring) library.

It sets up an automatic consistent hash ring across multiple nodes. The nodes
are discovered and health validated either over Memberlist's implementation of
the SWIM gossip protocol or via Sidecar. It manages adding and removing nodes
from the ring based on events from Memberlist or Sidecar.

In addition, the package provides some HTTP handlers and a Mux that can be
mounted anywhere in your application if you want to expose the hashring.

The Problem This Solves
-----------------------

If you're building a cluster of nodes and need to distribute data amongst them,
you have a bunch of choices. One good choice is to build a consistent hash and
back it with a list of cluster nodes. When distributing work/data/etc you can
then do a lookup against the hash to identify which node(s) should handle the
work. But now you need to maintain the list of nodes, and identify when they
come and go from the cluster, and other details of clustering that don't really
have much to do with the application you're trying to write.  Sometimes a
specialized load balancer can solve this problem for you. Sometimes you need to
write your own mechanism.

This is where ringman comes in. It maintains a consistent hash and the
clustering mechanism underneath it via one of two different provided backends.
It also offers an optional queryable web API so you or other services can
inspect the state of the cluster.

It takes one line of code to set up the cluster, and one line of code to
query it!

Memberlist Ring
---------------

If you just want to get information from the ring you can query it in your code
directly like:

```go
ring, err := ringman.NewDefaultMemberlistRing([]string{127.0.0.1}, "8000")
if err != nil {
    log.Fatalf("Unble to establish memberlist ring: %s", err)
}

println(ring.Manager.GetNode("mykey"))
```

The following would set up a Memberlist-backed consistent hash ring and serve
the node information over HTTP:

```go
ring, err := ringman.NewDefaultMemberlistRing([]string{127.0.0.1}, "8000")
if err != nil {
    log.Fatalf("Unble to establish memberlist ring: %s", err)
}

http.HandleFunc("/your_stuff", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })
http.Handle("/hashring/", http.StripPrefix("/hashring", ring.HttpMux()))
err = http.ListenAndServe(":8080", http.DefaultServeMux)
if err != nil {
	log.Fatalf("Unable to start HTTP server: %s", err)
}
```

We can then query it with:

```
$ curl http://localhost:8000/hashring/nodes
```

Which will return output like:

```
[
  {
    "Name": "ubuntu",
    "Addr": "127.0.0.1",
    "Port": 7946,
    "Meta": null,
    "PMin": 1,
    "PMax": 4,
    "PCur": 2,
    "DMin": 0,
    "DMax": 0,
    "DCur": 0
  }
]
```

Or we can find the node for a specific key like:

```
$ curl http://docker1:8000/hashring/nodes/get?key=somekey
```

Which returns output like:

```
{
  "Node": "ubuntu",
  "Key": "somekey"
}
```

### More About Memberlist
If you are going to set up the Memberlist ring, it may be helpful to read up on
[Memberlist](https://github.com/hashicorp/memberlist) and the [SWIM
paper](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf) that explains
roughly how the underlying algorithm works.

Sidecar Ring
------------

An alternate implementation backed by New Relic/Nitro's
[Sidecar](https://github.com/Nitro/sidecar) is available as well. Underneath
Sidecar also lies Memberlist, but Sidecar provides a lot of features on top of
it. This implementation of Ringman assumes it will be subscribed to incoming
Sidecar events on the listener port. See the Sidecar
[README](https://github.com/Nitro/sidecar) for how to set that up. Once that is
established, you can do the following:

```go
ring, err := ringman.NewSidecarRing("http://localhost:7777/api/state.json")
if err != nil {
    log.Fatalf("Unble to establish sidecar ring: %s", err)
}

println(ring.Manager.GetNode("mykey"))
```

