# mpdprxy

[![Build Status](https://travis-ci.org/felixb/mpdprxy.svg?branch=master)][3]

Proxy MPD connections to multiple servers.
The proxy makes it possible to build a multi room audio system with a bunch of raspberry pis running [mopidy][1].

# Install

Just compile the code with go and run it:

    go build mpdprxy

It's possible to [cross compile][2] the proxy to run on raspberry pi.

# Usage

    mpdprxy --port 6601 --hosts rsp0:6600,rsp1:6600 --http 8080

* `--port 6601` makes the proxy listen on port 6601 for incoming connections.
* `--hosts rsp0:6600,rsp1:6600` makes the proxy forward connections to hosts rsp0 and rsp1 using default MPD port.
* `--http 8080` starts a very simple http interface allowing to configure the proxy on runtime.

Every configuration change done via http will reset all connections.
It's necessary to make clients aware of changed backends.

# License

    Copyright 2014 Felix Bechstein

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.


[1]: https://github.com/mopidy/mopidy
[2]: http://dave.cheney.net/2013/07/09/an-introduction-to-cross-compilation-with-go-1-1
[3]: https://travis-ci.org/felixb/mpdprxy
