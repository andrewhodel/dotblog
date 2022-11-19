package main

import (
	"fmt"
	"os"
	"time"
	"net"
	"crypto/tls"
	"crypto/x509"
	"log"
	"io"
	"io/ioutil"
	"strings"
	"strconv"
	"bytes"
	"compress/zlib"
	"net/http"
	"github.com/andrewhodel/go-ip-ac"
	"path/filepath"
	"errors"
	"sort"
)

var updating_content = false
var categories map[string] []string
var new_categories map[string] []string
var posts_by_date map[string] time.Time
var new_posts_by_date map[string] time.Time
var titles map[string] string
var new_titles map[string] string
var short_posts map[string] string
var content map[string] string
var new_content map[string] string
var sending_content = 0
var ip_ac ipac.Ipac
var mime_types map[string] string

func parse_post(post_path string, p string) {
	// do not use as a go subroutine

	// displayed on recent posts
	var short_html = ""
	// displayed when post is viewed
	var full_html = "<div class=\"post\">"

	var title_string = ""
	var ts_string = ""
	var full_html_started = false

	// parse .blog file
	var newline_counter = 0
	var block_counter = 0
	var lines = strings.Split(p, "\n")
	for l := range(lines) {
		var line = lines[l]

		if (strings.Index(line, "//") == 0) {
			// skip comment
			continue
		}

		if (block_counter == 0) {
			// headers
			//fmt.Println("headers line", line)

			if (strings.Index(line, "date: ") == 0) {
				// parse date
				// unix timestamp, seconds since 1970
				date, err := strconv.ParseInt(strings.TrimPrefix(line, "date: "), 10, 64)
				if (err == nil) {
					var ts = time.Unix(date, 0)
					new_posts_by_date[post_path] = time.Unix(date, 0)
					ts_string = "<span class=\"unix_ts post_date\">" + strconv.FormatInt(ts.Unix(), 10) + "</span>"
				} else {
					fmt.Println("error parsing date for file:", post_path, err)
				}
			} else if (strings.Index(line, "categories: ") == 0) {
				var cats_str = strings.TrimPrefix(line, "categories: ")
				var cats = strings.Split(cats_str, ", ")

				for c := range cats {

					var cat = cats[c]
					new_categories[cat] = append(new_categories[cat], post_path)

				}

			} else if (strings.Index(line, "title: ") == 0) {
				var title = strings.TrimPrefix(line, "title: ")
				new_titles[post_path] = title
				title_string = "<span class=\"post_title\">" + title + "</span>"
				short_html += "<div class=\"recent_posts_entry\"><h1><a href=\"" + post_path + "\">" + title + "</a></h1></div>"
			}

		} else if (block_counter == 1) {
			// short html
			//fmt.Println("short html line", line)
			short_html += line
		} else if (block_counter == 2) {
			// full html
			//fmt.Println("full html line", line)
			if (full_html_started == false) {
				// put full html in post_content class
				full_html += title_string + ts_string + "<div class=\"post_content\">"
				full_html_started = true
			}
			full_html += line
		}

		if (block_counter < 2) {
			// newlines are not counted in the full post html
			if (len(line) == 0 || line == "\r") {
				// empty line or line with \r
				newline_counter += 1
			}
		}

		if (newline_counter == 2) {
			block_counter = block_counter + 1
			newline_counter = 0
		}

	}

	short_posts[post_path] = short_html
	new_content["url:/" + post_path] = full_html + "</div></div>"

	return

}

func content_loop() {

	if (sending_content != 0) {
		// wait for content send routines
		time.Sleep(time.Minute * 1)
		go content_loop()
		return
	}

	// check for updated content
	var update_content = false

	// check files in posts/
	// and update template
	err := filepath.Walk("posts", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if (path != "posts") {

			if (strings.Index(path, ".blog") != len(path) - 5) {
				return errors.New("not a .blog file: " + string(path))
			}

			//fmt.Println("path:", path, info.Size())

			var fc, rf_err = ioutil.ReadFile(path)
			if (content["url:/" + path] != string(fc) && rf_err == nil) {
				parse_post(string(path), string(fc))
				update_content = true
			}

		}

		return nil

	})

	if err != nil {
		fmt.Println("filepath.Walk error:", err)
	}

	// check index.html
	index_html, index_err := ioutil.ReadFile("main/index.html")
	if (index_err != nil) {
		fmt.Println("main/index.html does not exist")
		os.Exit(1)
	}

	// use deflate (zlib) compression
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(index_html))
	w.Close()

	if (b.String() != content["url:/"] || update_content == true) {

		// main/index.html was modified
		// or there are posts that are new or modified
		update_content = true

		new_content["header"] = ""
		new_content["footer"] = ""

		// create categories html
		var categories_html = ""

		// order new_categories by character
		srr := make([]string, 0)
		for k := range new_categories {
			srr = append(srr, k)
		}

		sort.Strings(srr)

		for k := range srr {
			for d := range new_categories {
				if (srr[k] == d) {
					//var posts_in_cat = new_categories[d]
					categories_html += "<a href=\"/categories/" + d + "\" class=\"categories_entry\">" + d + "</a>"
					break
				}
			}
		}

		// add all posts sorted by time to html blocks
		var short_posts_html = ""
		var post_titles_html = ""

		// order new_posts_by_date
		sr := make([]int, 0)
		for k := range new_posts_by_date {
			sr = append(sr, int(new_posts_by_date[k].Unix()))
		}

		sort.Ints(sr)

		// reverse the slice
		rev_sr := make([]int, 0)
		for k := range sr {
			_ = k

			// add the last entry to rev_sr
			rev_sr = append(rev_sr, sr[len(sr)-1])
			// remove the last entry from sr
			sr = sr[:len(sr)-1]

		}

		var count = 0
		for k := range rev_sr {
			for d := range new_posts_by_date {
				if (rev_sr[k] == int(new_posts_by_date[d].Unix())) {

					var post_path = d
					//var post_time = new_posts_by_date[d]

					if (count < 20) {

						// only place the most recent 20 posts in short_posts_html

						for p := range short_posts {

							if (post_path == p) {
								short_posts_html += short_posts[p]
								break
							}

						}

					}

					if (count < 40) {

						// only place the most recent 40 posts in post_titles_html

						for t := range new_titles {

							if (post_path == t) {
								post_titles_html += "<a href=\"/" + t + "\" class=\"post_titles_entry\">" + new_titles[t] + "</a>"
								break
							}

						}

					}

					count += 1

					break
				}
			}
		}

		// add sections to index_html
		var new_index_html = ""
		var header = ""
		var footer = ""
		var header_footer_flip = false
		var lines = strings.Split(string(index_html), "\n")
		for l := range(lines) {

			var line = lines[l]

			if (line == "<!-- ######categories###### -->") {

				// add all the categories
				lines[l] = categories_html

			} else if (line == "<!-- ######posts###### -->") {

				// add the most recent posts
				lines[l] = short_posts_html

				// stop adding to the header after this
				// to replace this segment with content if not index.html
				header_footer_flip = true

			} else if (line == "<!-- ######post_titles###### -->") {

				// add all posts
				lines[l] = post_titles_html

			}

			if (line != "<!-- ######posts###### -->") {

				// all lines except this one are added to the header and footer
				// and this line flips them

				if (header_footer_flip == false) {
					header += line
				} else {
					footer += line
				}

			}

			new_index_html += lines[l]

		}

		// use deflate (zlib) compression
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		w.Write([]byte(new_index_html))
		w.Close()
		new_content["url:/"] = b.String()

		// add categories and post_titles to header and footer
		header = strings.Replace(header, "<!-- ######categories###### -->", categories_html, 1)
		header = strings.Replace(header, "<!-- ######post_titles###### -->", post_titles_html, 1)
		footer = strings.Replace(footer, "<!-- ######categories###### -->", categories_html, 1)
		footer = strings.Replace(footer, "<!-- ######post_titles###### -->", post_titles_html, 1)

		new_content["header"] = header
		new_content["footer"] = footer

	}

	if (update_content == true) {

		// set updating_content to true
		updating_content = true

		// delete short_posts
		for l := range short_posts {
			delete(short_posts, l)
		}

		// delete categories 
		for l := range categories {
			delete(categories, l)
		}

		// delete posts_by_date
		for l := range posts_by_date {
			delete(posts_by_date, l)
		}

		// delete titles
		for l := range titles {
			delete(titles, l)
		}

		// delete content
		for l := range content {
			delete(content, l)
		}

		// update categories map
		for l := range new_categories {

			if (len(new_categories[l]) == 0) {
				// delete empty map value from categories
				delete(categories, l)
			} else {
				// copy new_categories map value to categories map
				categories[l] = new_categories[l]
			}

			// delete map value from new_categories
			delete(new_categories, l)

		}

		// update posts_by_date map
		for l := range new_posts_by_date {

			if (new_posts_by_date[l] == time.Time{}) {
				// delete empty map value from posts_by_date
				delete(posts_by_date, l)
			} else {
				// copy new_posts_by_date map value to posts_by_date map
				posts_by_date[l] = new_posts_by_date[l]
			}

			// delete map value from new_posts_by_date
			delete(new_posts_by_date, l)

		}

		// update titles map
		for l := range new_titles {

			if (new_titles[l] == "") {
				// delete empty map value from titles
				delete(titles, l)
			} else {
				// copy new_titles map value to titles map
				titles[l] = new_titles[l]
			}

			// delete map value from new_titles
			delete(new_titles, l)

		}

		// update content map
		for l := range new_content {

			//fmt.Println(l, new_content[l])

			if (new_content[l] == "") {
				// delete empty map value from content
				delete(content, l)
			} else {
				// copy new_content map value to content map
				content[l] = new_content[l]
			}

			// delete map value from new_content
			delete(new_content, l)

		}

		// set updating_content to false
		updating_content = false

	}

	time.Sleep(time.Minute * 1)

	go content_loop()

}

func handle_http_request(w http.ResponseWriter, r *http.Request) {

	// if changes to memory from files are processing, wait for the updated content map
	if (updating_content == true) {
		time.Sleep(time.Millisecond * 200)
		// try again
		handle_http_request(w, r)
		return
	}

	sending_content = sending_content + 1

	// add cache headers
	w.Header().Set("Cache-Control", "max-age=604800")

	if (r.URL.Path == "/") {

		// main view
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "deflate")
		io.WriteString(w, content["url:/"])

	} else if (strings.Index(r.URL.Path, "/categories/") == 0) {

		// get category
		var cat = strings.TrimPrefix(r.URL.Path, "/categories/")

		if (len(categories[cat]) > 0) {
			// exists
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, content["header"])

			var s = ""
			for c := range categories[cat] {
				var post_path = categories[cat][c]

				var title = get_post_title(post_path)
				var ts = strconv.FormatInt(get_post_ts(post_path), 10)

				s += "<div class=\"category_post_entry\"><a href=\"/" + post_path + "\" class=\"category_post_link\">" + title + "</a><span class=\"unix_ts category_post_date\">" + ts + "</span></div>"
			}

			io.WriteString(w, s + content["footer"])
		} else {
			// does not exist
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "not found")
		}

	} else if (strings.Index(r.URL.Path, "/posts/") == 0) {

		// a post
		if (content["url:" + r.URL.Path] == "") {
			// does not exist
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, "not found")
		} else {
			// exists
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, content["header"] + content["url:" + r.URL.Path] + content["footer"])
		}

	} else if (strings.Index(r.URL.Path, "/..") != -1) {

		// invalid URL, someone is trying to access a file they should not be trying to access
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "")

	} else {

		// a file accessed by the browser, included in the /main directory
		f, err := os.Open("main" + r.URL.Path)

		if (err != nil) {

			// file not found
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, "not found")
		} else {

			// get extension
			var ext_p = strings.Split(r.URL.Path, ".")
			var ext = ""
			if (len(ext_p) >= 2) {
				ext = ext_p[len(ext_p) - 1]
				w.Header().Set("Content-Type", mime_types[ext])
			}

			if (ext == "") {
				w.Header().Set("Content-Type", "application/octet-stream")
			}

			// send content
			for (true) {
				b := make([]byte, 1024)
				n, read_err := f.Read(b)
				if (read_err != nil) {
					// sent whole file or there was an error
					break
				}
				_ = n
				w.Write(b)
			}

			f.Close()

		}

	}

	sending_content = sending_content - 1

}

func get_post_title(post_path string) (string) {

	var title = ""
	for l := range titles {

		if (l == post_path) {
			title = titles[l]
			break
		}

	}

	return title

}

func get_post_ts(post_path string) (int64) {

	var date time.Time
	for l := range posts_by_date {
		if (l == post_path) {
			date = posts_by_date[l]
			break
		}
	}

	return date.Unix()

}

func timeago(t time.Time) (string) {
	// return time ago in readable format

	// first get seconds
	var ago = time.Now().Unix() - t.Unix()

	var s = "s"

	if (ago >= 60 * 60 * 24 * 365) {
		// get years
		ago = ago / (60 * 60 * 24 * 365)
		s = "y"
	} else if (ago >= 60 * 60 * 24 * 30) {
		// get months
		ago = ago / (60 * 60 * 24 *30)
		s = "m"
	} else if (ago >= 60 * 60 * 24) {
		// get days
		ago = ago / (60 * 60 * 24)
		s = "d"
	} else if (ago >= 60 * 60) {
		// get hours
		ago = ago / (60 * 60)
		s = "h"
	} else if (ago >= 60) {
		// get minutes
		ago = ago / 60
		s = "m"
	}

	return strconv.FormatInt(ago, 10) + s + " ago"

}

func main() {

	new_categories = make(map[string] []string)
	categories = make(map[string] []string)
	new_posts_by_date = make(map[string] time.Time)
	posts_by_date = make(map[string] time.Time)
	new_titles = make(map[string] string)
	titles = make(map[string] string)
	short_posts = make(map[string] string)
	new_content = make(map[string] string)
	content = make(map[string] string)

	// basic mime types
	mime_types = make(map[string] string)
	mime_types["txt"] = "text/plain"
	mime_types["html"] = "text/html"
	mime_types["jpeg"] = "image/jpeg"
	mime_types["jpg"] = "image/jpeg"
	mime_types["png"] = "image/png"
	mime_types["gif"] = "image/gif"
	mime_types["webp"] = "image/webp"
	mime_types["json"] = "application/json"
	mime_types["xml"] = "text/xml"
	mime_types["svg"] = "image/svg+xml"
	mime_types["js"] = "text/javascript"
	mime_types["css"] = "text/css"

	// update content first to include all existing content if server is running
	go content_loop()

	var port int64 = 444;

	tls_config, err := createServerConfig("./keys/server.ca-bundle", "./keys/server.crt", "./keys/server.key")
	if err != nil {
		fmt.Printf("tls config failed: %s\n", err.Error())
		os.Exit(1)
	}

	// listen on tcp socket
	ln, err := tls.Listen("tcp", ":" + strconv.FormatInt(port, 10), tls_config)
	if err != nil {
		fmt.Printf("listen failed: %s\n", err.Error())
		os.Exit(1)
	}
	defer ln.Close()

	// go-ip-ac
	ipac.Init(&ip_ac)

	// invoke the HTTPS server
	srv := &http.Server{
		// keep-alives are enabled by default
		IdleTimeout: 5 * time.Second,
		// this is required to find invalid TLS connections
		ErrorLog: httpLogger(),
		// no reason for this to be larger
		MaxHeaderBytes: 1500,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// slash is the catch all in golang

		// take the port number off the address
		var ip, port, iperr = net.SplitHostPort(r.RemoteAddr)
		_ = port
		_ = iperr

		if (ipac.TestIpAllowed(&ip_ac, ip) == false) {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, "")
			return
		}

		//fmt.Println("request", ip, r.URL.Path)

		handle_http_request(w, r)

	})

	https_err := srv.Serve(ln)
	if https_err != nil {
		fmt.Println("Error starting HTTPS Server")
		fmt.Println(https_err)
		os.Exit(1)
	}

	fmt.Println("HTTPS Service Started on Port " + strconv.FormatInt(port, 10))

}

// a Writer and a Logger are needed for normal http logging
type learnSwiftStructsTooWriter struct {
}

func (e learnSwiftStructsTooWriter) Len() (int) {

	// return length of existing buffer
	// Logger requires this
	return 0

}

func (e learnSwiftStructsTooWriter) Write(p []byte) (int, error) {

	//fmt.Printf("%s\n", p)

	// create a failed TLS handshake with
	// nc domain.tld 443 </dev/null
	if (bytes.Index(p, []byte("TLS handshake error")) > -1) {
		// get the ip address from
		// http: TLS handshake error from 77.35.198.143:63185: EOF
		// as sent from c.server.logf("http: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), err) in the Go source
		var s = bytes.Split(p, []byte(" "))
		//fmt.Println(len(s))
		if (len(s) >= 6) {
			// string matches pattern
			//fmt.Println(string(s[5]))
			var ip_info = strings.Split(string(s[5]), ":")
			//fmt.Println(len(ip_info))
			if (len(ip_info) == 3) {
				// valid by counting
				var ip = ip_info[0]
				//fmt.Println(ip)

				// test the ip to increment the connection counters for this ip
				ipac.TestIpAllowed(&ip_ac, ip)

			}
		}
	}

	// return (number of bytes written from p, err)
	return len(p), nil

}

func httpLogger() *log.Logger {

	buf := learnSwiftStructsTooWriter{}
	// create a logger that uses a custom writer
	// and no prefix
	logger := log.New(buf, "", 0)

	return logger

}

func createServerConfig(ca, crt, key string) (*tls.Config, error) {

	roots := x509.NewCertPool()

	caCertPEM, err := ioutil.ReadFile(ca)
	if err != nil {
		fmt.Println("CA file not required, but not validated either:", err)
	} else {

		ok := roots.AppendCertsFromPEM(caCertPEM)
		if !ok {
			panic("failed to parse root certificate")
		}

	}

	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:	tls.VerifyClientCertIfGiven,
		ClientCAs:    roots,
	}, nil
}
