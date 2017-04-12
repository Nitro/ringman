Ringman
=======

This is a consistent hash ring implementation backed by [our fork of
Hashicorp's Memberlist library](https://github.com/Nitro/memberlist) and the
[hashring](https://github.com/serialx/hashring) library.

It sets up an automatic consistent hash ring across multiple nodes. The nodes
are discovered and health validated over Memberlist's implementation of the
SWIM gossip protocol. It manages adding and removing nodes from the ring based
on events from Memberlist.

In addition, the package provides some HTTP handlers and a Mux that can be
mounted anywhere in your application if you want to expose the hashring.

If you just want to get information from the ring you can query it in your code
directly like:

```
ring, err := ringman.NewDefaultMemberlistRing([]string{127.0.0.1})
if err != nil {
    log.Fatalf("Unble to establish memberlist ring: %s", err)
}

println(ring.Manager.GetNode("mykey"))
```

The following would set up a Memberlist-backed consistent hash ring and serve
the node information over HTTP:

```go
ring, err := ringman.NewDefaultMemberlistRing([]string{127.0.0.1})
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

