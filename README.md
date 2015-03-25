rb-gateway
==========
A service for managing your repositories.

Install
-------
Instructions to install rb-gateway in the `$GOPATH/src/` directory from the master branch:

    $ go get -d github.com/reviewboard/rb-gateway
    $ cd github.com/reviewboard/rb-gateway
    $ mv sample_config.json config.json
    $ go get
    $ go install

    Modify config.json to point to your repositories.

To start the server on localhost:8888:

    go run *.go

Dependencies
------------
This project depends on [git2go](https://github.com/libgit2/git2go/) and
[Gorilla web toolkit's mux package](http://www.gorillatoolkit.org/pkg/mux).

rb-gateway will autofetch dependencies with `go get`.

Testing
-------
`go test` will run automated tests against rb-gateway.

License
-------
The MIT License (MIT)

Copyright (c) 2015 Review Board

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.