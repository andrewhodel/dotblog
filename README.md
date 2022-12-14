# .blog

Manage posts with files.

![Make new posts with .blog files.](/readme_resources/post.blog.png)

Designs are HTML/CSS/Javascript.

![Write designs in HTML/CSS/JavaScript.](/readme_resources/index.html.png)

Content is updated every minute based on file system changes, without restarting the server.

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

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run dotblog_server.go` to run in the foreground.

`sudo GOPATH=/home/ec2-user/go GO111MODULE=off go run dotblog_server.go > /dev/null 2>&1 &` to run in the background.

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
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
```

## upgrading

`git pull` will upgrade .blog; `keys/`, `main/` and `posts/` are not modified.

# donate

## Bitcoin
BTC 39AXGv2up1Yk5QNeLHfQra815jaYv9HcJk

## Credit Card
[![Paypal Donation](/readme_resources/paypal_donate_button.gif "Paypal Donation")](https://www.paypal.com/donate/?hosted_button_id=5XCWCGPC2FBU6)

## Paypal by QR Code
![Paypal QR Donation](/readme_resources/paypal_donate_qr.png "Paypal QR Donation")

# verification

The OP_RETURN data in this BTC transaction provides btc-blockchain-copy-count checksum verification of this repository, the associated github account, the files and the commit dates.

https://blockstream.info/tx/9d014787b37a535085db55680b89b37cfc939ac61959e920051ac2720d5a3314?expand

# license

dotblog uses the MIT License
