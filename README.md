# .blog

A file based blog.

Designs are simple again.

![Write designs in HTML/CSS/JavaScript.](/screenshots/index.html.png)

Posts are files.

![Make new posts with .blog files.](/screenshots/post.blog.png)

# installation

```
git clone https://github.com/andrewhodel/dotblog
cd dotblog
cp -r default_html main
mkdir posts
mkdir keys
```

1. Create the `keys` directory and place your TLS keys in it.

* server.ca-bundle
* server.crt
* server.key

2. Install the required Go modules.

```
GO111MODULE=off go get -u github.com/andrewhodel/go-ip-ac
```

3. Run the server.

`sudo` allows `iptables` permission.

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run blog.go` to run in the foreground.

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run blog.go > /dev/null 2>&1 &` to run in the background.

## style

Edit the files in `main/`, it's HTML, CSS and JavaScript.

Only `index.html` is required.

## .blog file format

These files are placed in `posts/`, read `post_template.blog` and copy it to a new file in `posts/` to create a new post.

The file names create unique urls that will be indexed by search engines.

### self signed certificate

You can create self signed certificates.

```
cd keys/
openssl req -new -subj "/C=US/ST=Utah/CN=localhost" -newkey rsa:2048 -nodes -keyout server.key -out server.csr
openssl x509 -req -days 365 -in localhost.csr -signkey server.key -out server.crt
```

## upgrading

`git pull` will upgrade .blog; `keys/`, `main/` and `posts/` are not modified.

# donate

## Bitcoin
BTC 39AXGv2up1Yk5QNeLHfQra815jaYv9HcJk

## Credit Card
[![Paypal Donation](/img/paypal_donate_button.gif "Paypal Donation")](https://www.paypal.com/donate/?hosted_button_id=5XCWCGPC2FBU6)

## Paypal by QR Code
![Paypal QR Donation](/img/paypal_donate_qr.png "Paypal QR Donation")
