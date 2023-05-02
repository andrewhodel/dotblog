# .blog

Manage posts with files.

![Make new posts with .blog files.](/readme_resources/post.blog.png)

Designs are HTML/CSS/Javascript.

![Write designs in HTML/CSS/JavaScript.](/readme_resources/index.html.png)

Content is updated every minute based on file system changes, without restarting the server.

# Installation

```
git clone https://github.com/andrewhodel/dotblog
cd dotblog
cp -r default_html main
mkdir posts
cp config_sample.json config.json
```

1. Edit config.json

* set the paths to your TLS keys or place the key data in config.json
* set the fqdn (fully qualified domain name) of the server
* set the ipacModuleDirectory or the go-ip-ac Go Module must be in $HOME/go/src/github.com/andrewhodel/go-ip-ac

2. Install the required Go modules.

```
GO111MODULE=off go get -u github.com/andrewhodel/go-ip-ac
```

3. Run the server.

`sudo` allows `iptables` permission.

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run dotblog_server.go` to run in the foreground.

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run dotblog_server.go > /dev/null 2>&1 &` to run in the background.

## Style

Edit the files in `main/`, it's HTML, CSS and JavaScript.

Only `index.html` is required.

## .blog File Format

These files are placed in `posts/`, read `post_template.blog` and copy it to a new file in `posts/` to create a new post.

The file names create unique urls that will be indexed by search engines.

### Self Signed Certificate

You can create self signed certificates.

```
mkdir keys
cd keys/
openssl req -new -subj "/C=US/ST=Utah/CN=localhost" -newkey rsa:2048 -nodes -keyout server.key -out server.csr
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
```

## /path HTTP requests require a trailing `/` or `<base>` in the HTML

Included HTML content with relative paths after a subdirectory in main will not work without a trailing `/` or `<base>`.

https://developer.mozilla.org/en-US/docs/Web/HTML/Element/base

A slower method is a 3xx redirect to `/path/index.html` using the response headers to the `/path` request, it requires 2 requests.

## Upgrading

`git pull` will upgrade .blog

## go-ip-ac Firewall Output

Output the go-ip-ac information to stdout by sending SIGUSR1 to the process.

```
[ec2-user@ip dotblog]$ ps aux |grep dotblo
root      3238  0.0  0.7 239820  7140 ?        S    02:09   0:00 sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run dotblog_server.go
root      3239  0.0  1.3 1161948 13388 ?       Sl   02:09   0:00 go run dotblog_server.go
root      3277  0.0  1.4 1229652 14596 ?       Sl   02:09   0:04 /tmp/go-build1939384177/b001/exe/dotblog_server
ec2-user 27352  0.0  0.0 119428   940 pts/1    S+   06:41   0:00 grep --color=auto dotblo
[ec2-user@ip dotblog]$ sudo kill -s SIGUSR1 3277
```

```
go-ip-ac IP information:
{CleanupLoopSeconds:60 BlockForSeconds:86400 BlockIpv6SubnetsGroupDepth:4 BlockIpv6SubnetsBreach:40 WarnAfterNewConnections:80 WarnAfterUnauthedAttempts:5 BlockAfterNewConnections:1700 BlockAfterUnauthedAttempts:30 NotifyAfterAbsurdAuthAttempts:20 NotifyClosure:<nil> Purge:false LastCleanup:1683010756 LastNotifyAbsurd:1682924296 NextNotifyBlockedIps:[] NextNotifyAbsurdIps:[] Ips:[{Addr:8.8.8.8 Authed:false Warn:false Blocked:false LastAccess:1683010768 LastAuth:0 UnauthedNewConnections:1 UnauthedAttempts:0 AbsurdAuthAttempts:0} {Addr:8.8.8.7 Authed:false Warn:false Blocked:false LastAccess:1683010733 LastAuth:0 UnauthedNewConnections:1 UnauthedAttempts:0 AbsurdAuthAttempts:0}] Ipv6Subnets:[] TotalCount:2 BlockedCount:0 WarnCount:0 BlockedSubnetCount:0 ModuleDirectory:/home/ec2-user/go/src/github.com/andrewhodel/go-ip-ac NeverBlock:false}

{Addr:8.8.8.8 Authed:false Warn:false Blocked:false LastAccess:1683010768 LastAuth:0 UnauthedNewConnections:1 UnauthedAttempts:0 AbsurdAuthAttempts:0}
{Addr:8.8.8.7 Authed:false Warn:false Blocked:false LastAccess:1683010733 LastAuth:0 UnauthedNewConnections:1 UnauthedAttempts:0 AbsurdAuthAttempts:0}
```

# Donate

## Bitcoin
BTC 39AXGv2up1Yk5QNeLHfQra815jaYv9HcJk

## Credit Card
[![Paypal Donation](/readme_resources/paypal_donate_button.gif "Paypal Donation")](https://www.paypal.com/donate/?hosted_button_id=5XCWCGPC2FBU6)

## Paypal by QR Code
![Paypal QR Donation](/readme_resources/paypal_donate_qr.png "Paypal QR Donation")

# Verification

The OP_RETURN data in this BTC transaction provides btc-blockchain-copy-count checksum verification of this repository, the associated github account, the files and the commit dates.

https://blockstream.info/tx/9d014787b37a535085db55680b89b37cfc939ac61959e920051ac2720d5a3314?expand

# License

dotblog uses the MIT License
