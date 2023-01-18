/*
Copyright 2022 Andrew Hodel
andrewhodel@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"fmt"
	"os"
	"time"
	"net"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log"
	"io"
	"io/ioutil"
	"strings"
	"strconv"
	"bytes"
	"net/http"
	"github.com/andrewhodel/go-ip-ac"
	"path/filepath"
	"sort"
)

type Config struct {
	SslKey				string	`json:"sslKey"`
	SslCert				string	`json:"sslCert"`
	SslCa				string	`json:"sslCa"`
	LoadCertificatesFromFiles	bool	`json:"loadCertificatesFromFiles"`
	Fqdn				string	`json:"fqdn"`
	Port				int64	`json:"port"`
	RedirectFromDefaultHttpPort	bool	`json:"redirectFromDefaultHttpPort"`
	IpacModuleDirectory		string	`json:"ipacModuleDirectory"`
}

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
var config Config
var mime_types map[string] string

func parse_post(post_path string, p string) {
	// do not use as a go subroutine

	// displayed on recent posts
	var short_html = ""
	// displayed when post is viewed
	var full_html = "<div class=\"post\">"

	var title_string = ""
	var ts_string = ""
	var categories_string = ""
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

					// add to new_categories
					new_categories[cat] = append(new_categories[cat], post_path)

					// add to categories_string as html element to be displayed when the full post is viewed
					categories_string += "<a href=\"/categories/" + cat + "\">" + cat + "</a>"

				}

			} else if (strings.Index(line, "title: ") == 0) {

				// get the title from the header line
				var title = strings.TrimPrefix(line, "title: ")

				// add the title to new_titles
				new_titles[post_path] = title

				// store the title string
				title_string = "<span class=\"post_title\">" + title + "</span>"

				// create the start of short_html
				// with the unique strings that represent the positions of these blocks
				short_html += "<div class=\"recent_posts_entry\"><a class=\"recent_post_title\" href=\"" + post_path + "\">" + title + "</a><span class=\"unix_ts recent_post_date\"><!--######rp_ts######--></span><div class=\"recent_post_categories\"><!--######rp_cats######--></div><div class=\"recent_post_content\">" + "\n"

			}

		} else if (block_counter == 1) {
			// short html

			//fmt.Println("short html line", line)
			short_html += line + "\n"

		} else if (block_counter == 2) {
			// full html
			//fmt.Println("full html line", line)

			if (full_html_started == false) {

				// finish tags in short_html
				short_html += "</div></div>" + "\n"

				var rp_ts = strconv.FormatInt(get_post_ts(post_path, true), 10)
				var rp_cats = ""

				for c := range new_categories {

					var cat = new_categories[c]

					for l := range cat {
						if (cat[l] == post_path) {
							// add to rp_cats
							rp_cats += "<a href=\"/categories/" + c + "\">" + c + "</a>"
							break
						}
					}

				}

				// replace the unique strings that represent the positions of these blocks
				short_html = strings.Replace(short_html, "<!--######rp_ts######-->", rp_ts, 1)
				short_html = strings.Replace(short_html, "<!--######rp_cats######-->", rp_cats, 1)

				// put full html in post_content class
				full_html += title_string + ts_string + "<div class=\"post_categories\"><span class=\"post_categories_title\">Categories</span>" + categories_string + "</div>" + "<div class=\"post_content\">"
				full_html_started = true

			}

			full_html += line + "\n"

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
				//fmt.Println("not a .blog file: " + string(path))
				return nil
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

	if (string(index_html) != content["url:/"] || update_content == true) {

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
					header += line + "\n"
				} else {
					footer += line + "\n"
				}

			}

			new_index_html += lines[l] + "\n"

		}

		new_content["url:/"] = new_index_html

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
		io.WriteString(w, content["url:/"])

	} else if (strings.Index(r.URL.Path, "/categories/") == 0) {

		// get category
		var cat = strings.TrimPrefix(r.URL.Path, "/categories/")

		if (len(categories[cat]) > 0) {
			// exists
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, content["header"])

			var s = "<span class=\"category_title\">" + cat + "</span>"
			for c := range categories[cat] {
				var post_path = categories[cat][c]

				var title = get_post_title(post_path)
				var ts = strconv.FormatInt(get_post_ts(post_path, false), 10)

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

func get_post_ts(post_path string, use_new bool) (int64) {

	var from = posts_by_date
	if (use_new == true) {
		// use new posts by date map
		from = new_posts_by_date
	}

	var date time.Time
	for l := range from {
		if (l == post_path) {
			date = from[l]
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

	// set the module directory for ipac
	ip_ac.ModuleDirectory = config.IpacModuleDirectory

	// update content first to include all existing content if server is running
	go content_loop()

	// read the configuration file
	cwd, cwd_err := os.Getwd()
	if (cwd_err != nil) {
		fmt.Println(cwd_err)
		os.Exit(1)
	}
	config_file_data, err := ioutil.ReadFile(cwd + "/config.json")

	if (err != nil) {
		fmt.Printf("Error reading configuration file ./config.json (" + cwd + "/config.json): %s\n", err)
	}

	config_json_err := json.Unmarshal(config_file_data, &config)
	if (config_json_err != nil) {
		fmt.Printf("Error decoding ./config.json: %s\n", config_json_err)
		os.Exit(1)
	}

	var cert tls.Certificate
	var cert_err error
	if (config.LoadCertificatesFromFiles == true) {
		cert, cert_err = tls.LoadX509KeyPair(config.SslCert, config.SslKey)
	} else {
		cert, cert_err = tls.X509KeyPair([]byte(config.SslCert), []byte(config.SslKey))
	}

	if cert_err != nil {
		fmt.Printf("did not load TLS certificates: %s\n", cert_err)
		os.Exit(1)
	}

	tls_config := tls.Config{Certificates: []tls.Certificate{cert}, ClientAuth: tls.VerifyClientCertIfGiven, ServerName: config.Fqdn}

	// listen on tcp socket
	ln, err := tls.Listen("tcp", ":" + strconv.FormatInt(config.Port, 10), &tls_config)
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

	fmt.Println("HTTPS Service Starting on Port " + strconv.FormatInt(config.Port, 10))

	https_err := srv.Serve(ln)
	if https_err != nil {
		fmt.Println("Error starting HTTPS Server")
		fmt.Println(https_err)
		os.Exit(1)
	}

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
