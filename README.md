The website of Joel H. W. Weinberger, Go Edition!

Pre-build setup:
* Install Bower, for dependency installation: http://bower.io
* Install [Crisper](https://github.com/PolymerLabs/crisper), for compiling Polymer
  Components to CSP friendly forms, by running `sudo npm -g install
  https://github.com/PolymerLabs/crisper/archive/v2.0.1.tar.gz`.
* Run `bower install` to install dependencies.
* Run `./tools/setup-components.sh` to create external scripts for CSP. Make sure to
  do this any time you run `bower install`.
* Install go: https://golang.org/doc/install
* Make sure you have `$GOPATH` environment variable set to a location that files
  can be downloaded to, for example `/home/user/gocode`.
* Set `$GOBIN` to `$GOPATH/bin`.
* If you're using TLS, create a `cert_config.json` file, with two keys,
  `PrivateKey` and `FullChain` whose values are the path to the private key and
  the path to the full cert chain. Usually these will go in ./cert, which is
  already in the .gitignore file.
* Finally, run `go get` to get all remote packages for the Go build.

To build and run:
* Run `go install`.
* To directly run the server, you can try `go run server.go`, but this sometimes
  results in an error (for reasons I'm still debugging). In that case, run `go
  build server.go` then `./server`.
* By default, the server runs HTTPS on port 8443 and the HTTP (for redirects
  only) on port 8080. Use the options `--https-port=xxx` and `--http-port=yyy`
  to change the HTTPS and HTTP ports to `xxx` and `yyy`, respectively.
* If you want to use privileged ports on Linux (e.g. ports 443 and 80 for HTTPS
  and HTTP, respectively), you need to either:
  * Run the server as root (very bad)
  * Redirect the privileged ports to the server on non-privileged ports
	(complicated)
  * Give the program privileged port capabilities while still running as an
	unprivileged user (IMO easiest).
  Fortunately, the last option is finally possible on Linux from kernel version
  2.6.24 onward. To do so, compile the server, then, as root, run `setcap
  'cap_net_bind_service=+ep' /path/to/server`. This should allow the executable
  to bind to privileged ports without any other escalation of privileges. You
  can also just run `sudo ./tools/set-privileged-ports-cap server` to set the
  privilege on the server file.

Recommended Linux system service setup:
* Add `go-server.service` to `/etc/init.d/go-server`. Make sure to replace the
  variable values in the file with the actual desired values.
* Run `sudo update-rc.d go-server defaults` to install the service. Now `service
  go-server {start,stop,restart}` should be usable.
* Install monit (see https://mmonit.com/monit/ for documentation) for process
  monitoring and automatic service restart.
* Copy `go-server.monit` to `/etc/monit/conf.d/`. Make sure it is owned by root
  and has permissions 600. Note to manually replace the pidfile location with
  the desired actual location of the pidfile.
* You can just startup the server and monit manually at this point (`sudo
  service go-server start` and `sudo monit`, respectively), or you can reboot
  once to verify that they start on boot.

The static/img/lock.ico favicon is used under a Creative Commons
Attribution-Share Alike 3.0 Unported license, courtesy of Wikimedia user
Urutseg, converted from: http://commons.wikimedia.org/wiki/File:Crypto_stub.svg

The photo static/img/joel-weinberger-headshot.jpg is used courtesy of Steve
Hanna (http://www.vividmachines.com).

serviceworker-cache-polyfill.js is taken from
https://github.com/coonsta/cache-polyfill under an Apache v2.0 license.
