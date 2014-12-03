## Build and Install (Ubuntu Only)

```sh
git clone https://github.com/BitolaCo/bolt.git
cd bolt
sudo mkdir /etc/bolt
sudo cp config.sample.json /etc/bolt/config.json
sudo go build -o /usr/bin/local/bolt server/main.go
sudo mv bolt.conf /etc/init.d
```

The bolt.conf file in the 

## Configuration

If you followed the instructions above, the config file
will now be at `/etc/bolt/config.json`, it's default location.

You can run it from the command line, too, with the `--config` flag:

`bolt --config=/custom/config/dir/config.json`

A sample configuration file looks like this:

```json
{
  "hosts": {
    "127.0.0.1": "upload.wikimedia.org",
  },
  "storage": "", // Path to store the cached files in. Defaults to system temp directory.
  "ttl": 1, // Time to live of cached files, in minutes.
  "listen": ":8000", // Address to bind/listen on. Leave off the ip address to listen on 0.0.0.0
  "quality": 70, // Quality of resized JPEG's
  "colors": 256 // # Colors in resampled gif's
}
```

Most important part here is probably the `hosts` and `listen` options.
Hosts matches a reqest host name (the key) to an upstream server (where to fetch the images from).
In the example above, `"127.0.0.1": "upload.wikimedia.org"` means requests to 127.0.0.1 would be
mapped to the corresponding file on `upload.wikimedia.org`

## Usage

Any of the following ways are valid to access scaled images. In the examples
below, they would all result in an image that's 100px wide, with the height
being relative to the width.

`yourserver.com/100/some/file/photo.png`
`yourserver.com/some/file/photo.png/100`
`yourserver.com/some/file/photo.png?w=100`

To fetch the original, unmodified image, just access it directly:

`yourserver.com/some/file/photo.png/100`


## Using In HTML

For the automatic scaling, first include the built-in JavaScript file.

## Roadmap

1. Add support for TLS connections
2. Allow remote configuration files via http/https