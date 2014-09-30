package main

import (
    "bufio"
    "encoding/json"
    "flag"
    "fmt"
    "html/template"
    "io"
    "net"
    "net/http"
    "log"
    "strings"
)

var (
    templIndex = template.Must(template.New("index").Parse(templIndexStr))
    servers []Server
    connections = make([]net.Conn, 0, 8)
)

type Server struct {
    Host string
    Active bool
    Default bool
}

func update_default() {
    found := -1
    for i := range servers {
        s := &servers[i]
        if s.Active && s.Default {
            found = i
            break
        }
    }

    for i := range servers {
        s := &servers[i]
        if s.Active && (found < 0 || found == i) {
            log.Printf("set default server: %s\n", s.Host)
            s.Default = true
            found = i
        } else {
            s.Default = false
        }
    }
}

func set_default(idx int) {
    for i := range servers {
        s := &servers[i]
        if s.Active && i == idx {
            log.Printf("set default server: %s\n", s.Host)
            s.Default = true
        } else {
            s.Default = false
        }
    }
    // set any default if idx was not found
    update_default()
}

func forwardConnection(input net.Conn, outputs []net.Conn) {
    log.Printf("new forwarder\n")
    connections = append(connections, input)
    r := bufio.NewReader(input)
    for {
        line, err := r.ReadString('\n')
        if err != nil {
            // ignore errors for now
            // log.Printf("error reading data: %s\n", err)
            break
        }
        for _, output := range outputs {
            if output != nil {
                _, err := io.WriteString(output, line)
                if err != nil {
                    // ignore errors for now
                    // log.Printf("error writing data: %s\n", err)
                }
            }
        }
    }
    for i := range connections {
        if connections[i] == input {
            connections = append(connections[:i], connections[i+1:]...)
            break
        }
    }
    log.Printf("closed forwarder\n")
    log.Printf("#forwarder=%d\n", len(connections))
}

func handleConnection(input net.Conn) {
    l := len(servers)
    var first net.Conn
    outputs := make([]net.Conn, l)
    for i := range servers {
        s := &servers[i]
        // connect to every child
        if !s.Active {
            continue
        }
        output, err := net.Dial("tcp", s.Host)
        if err != nil {
	          log.Printf("error opening connection to host %s\n", s.Host)
            outputs[i] = nil
        } else {
            log.Printf("connected to host %s\n", s.Host)
            outputs[i] = output
            if first == nil {
                // forward output of first child
                first = output
            }
        }
    }

    if first != nil {
        // forward input to every child
        go forwardConnection(input, outputs)
        // forward output of first child
        inputs := []net.Conn {input}
        go forwardConnection(first, inputs)
    }
}

func close_connections() {
    for i := range connections {
        connections[i].Close()
    }
}

func listen(port int) {
    log.Printf("starting socket on port %d\n", port)
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        log.Fatalf("error listening on socket %s\n", err)
    }
    for i := range servers {
        s := &servers[i]
        log.Printf("forwarding to host %s\n", s.Host)
    }
    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Printf("error accepting socket %s\n", err)
            continue
        }
        go handleConnection(conn)
    }
}

func ServeIndex(w http.ResponseWriter, req *http.Request) {
    log.Printf("serve: %s\n", req.URL.Path)

    if req.URL.Path == "/style.css" {
      io.WriteString(w, styleCss)
      return
    }

    if req.URL.Path != "/" {
        http.NotFound(w, req)
        return
    }

    // parse params
    req.ParseForm()
    if req.Form != nil && len(req.Form) > 0 {
        for i := range servers {
            s := &servers[i]
            param_active := fmt.Sprintf("active[%d]", i)
            param_default := fmt.Sprintf("default[%d]", i)
            v := req.FormValue(param_active)
            if v == "1" {
                log.Printf("set active: %s\n", s.Host)
                s.Active = true
                v := req.FormValue(param_default)
                if v == "1" {
                    set_default(i)
                }
            } else {
                log.Printf("set inactive: %s\n", s.Host)
                s.Active = false
                s.Default = false
            }
        }
        update_default()
        close_connections()
    }

    // output
    if req.FormValue("out") == "json" {
        b, err := json.Marshal(servers)
        if err != nil {
            log.Printf("error printing json: %s", err)
            http.Error(w, "JSON marshalling failed", http.StatusInternalServerError)
        }
        w.Write(b)
    } else {
        templIndex.Execute(w, servers)
    }
}

func httpd(port int) {
    log.Printf("starting httpd on port %d\n", port)
    addr := fmt.Sprintf(":%d", port)
    http.Handle("/", http.HandlerFunc(ServeIndex))
    err := http.ListenAndServe(addr, nil)
    if err != nil {
        log.Fatal("error starting httpd:", err)
    }
}

func main() {
    var port = flag.Int("port", 6601, "Port to listen for connections")
    var hosts_string = flag.String("hosts", "", "Hosts the proxy connects to, separated by ','")
    var http_port = flag.Int("http", -1, "Start http server for managing the connections")
    flag.Parse()

    // hosts
    hosts_array := strings.Split(*hosts_string, ",")
    l := len(hosts_array)
    servers = make([]Server, l)
    for i, h := range hosts_array {
        var p string
        if strings.Contains(h, ":") {
            t := strings.Split(h, ":")
            h = strings.Trim(t[0], " ")
            p = strings.Trim(t[1], " ")
        } else {
            h = strings.Trim(h, " ")
            p = "6600"
        }
        h = fmt.Sprintf("%s:%s", h, p)
        servers[i] = Server{h, true, false}
    }
    update_default()

    // http
    if *http_port > 0 {
        go httpd(*http_port)
    }

    listen(*port)
}

const templIndexStr = `<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>mpd proxy</title>
<link rel="stylesheet" type="text/css" href="style.css">
</head>
<body>
<form id="content">
<ul>
{{$servers := .}}
{{range $i, $e := $servers }}
  <li>
    <div>
    <p class="left">{{$e.Host}}</p>
    <p class="right">
    <span class="margin-left">
    active <input name="active[{{$i}}]" type="checkbox" value="1" {{if $e.Active}} checked="checked" {{end}}/>
    </span>
    <span class="margin-left">
    default <input name="default[{{$i}}]" type="checkbox" value="1" {{if $e.Default}} checked="checked" {{end}}/>
    </span>
    </p>
    </div>
    <div class="clear separator"></div>
  </li>
{{end}}
</ul>
<div class="right">
<input type="submit" value="update"/>
</div>
<div class="clear"></div>
</form>
</body>
</html>
`

const styleCss = `
html {
  height: 100%;
  width: 100%;
}

body {
  height: 100%;
  padding: 0px;
  text-align: center;
}

.margin-left {
  margin-left: 15px;
}

#content {
  max-width: 400px;
  margin: 0px auto;
  text-align: left;
  padding: 15px;
  border: 1px dashed #333;
  background-color: #eee;
}

.left {
  margin: 0px;
  float: left;
}

.right {
  margin: 0px;
  float: right;
}

.clear {
  height: 0px;
  clear: both;
}

.separator {
  height: 15px;
}

ul {
  padding: 0px;
  list-style-type: none;
}
`
