A Golang powered image caching and resizing reverse proxy.
It supports automatic, on-the-fly image resizing, and caches the results 
for best performance.

## Usage

Any of the following ways are valid to access scaled images. In the examples
below, they would all result in an image that's 100px wide, with the height
being relative to the width.

`yourserver.com/100/some/file/photo.png`
`yourserver.com/some/file/photo.png?w=100`

To fetch the original, unmodified image, just access it directly:

`yourserver.com/some/file/photo.png `

## Build and Install (Ubuntu Only)

```sh
git clone https://github.com/BitolaCo/proximity.git
cd proximity
sudo mkdir -p /etc/proximity/ssl
sudo cp config/config.json /etc/proximity/config.json
sudo go build -o /usr/bin/local/proximity main.go
sudo cp config/proximity.conf /etc/init
```

## Configuration

If you followed the instructions above, the config file
will now be at `/etc/proximity/config.json`, it's default location.

You can run it from the command line, too, with the `--config` flag:

`proximity --config=/custom/config/dir/config.json`

A sample configuration file looks like this:

```json
{
  "hosts": {
    "127.0.0.1": "upload.wikimedia.org",
  },
  "storage": "",
  "ttl": 1,
  "listen": ":4000",
  "quality": 70, 
  "colors": 256
}
```

Here's a rundown of each variable:

- `hosts`: A key/value pair that matches the incoming address (i.e. `yourserver.com`) with the remote address (i.e. `upload.wikimedia.org`)
- `storage` Path to store the cached files in. Defaults to system temp directory (maybe `/tmp`, who knows?)
- `ttl` Time to live of cached files, in minutes.
- `listen` Address to bind/listen on. Leave off the ip address to listen on 0.0.0.0
- `quality` Quality of resized JPEG's. Applies to jpegs only.
- `colors` Colors in resampled gif's. Applies to gif's only.


Most important part here is probably the `hosts` and `listen` options.
Hosts matches a reqest host name (the key) to an upstream server (where to fetch the images from).
In the example above, `"127.0.0.1": "upload.wikimedia.org"` means requests to 127.0.0.1 would be
mapped to the corresponding file on `upload.wikimedia.org`

## Production configuration

Changes are, you'll want to use it with a proxy setup, though it's strictly not necessary as
it does support full, valid SSL connections if desired.

#### Nginx

```
# For Nginx
location / {
    proxy_pass        http://localhost:4000/;
}
```

#### Apache
```
ProxyPass / http://localhost:4000
ProxyPassReverse / http://localhost:4000
```

## Using In HTML

For the automatic scaling, a JavaScript package is being tested. The source is in the
`assets` directory of the repository.

The javascript file can be included by adding a script tag to the fixed location on
the Proximity server. The server will set the correct parameters automatically for you:

```html
<script src="//yourserver.com/script"></script>
```

The resizing assumes the images will fit the parent container. As such, 
you'll need to make sure the images have a the following css set up:

```css
.fluid {
  display: block;
  width: 100%;
  max-width: 100%;
}
```

The default class for the images is `.fluid`, although you can change that by passing the `c=className` to
the server when including the script tag mentioned above:

```html
<script src="//yourserver.com/script?c=my-cool-class"></script>
```

In the example above, the JavaScript would then to it's work with any element with the class name `.my-cool-class`
instead of the default class.

## Roadmap

- Allow timed/remote reload of configuration
- Remove out-dated cache automatically
- Write tests
- Lock down sizing to a predefined set of widths (Small differences handled via CSS and 100% width, would also support srcset?)
- Generate all sizes automatically on first visit.
- Logging to system log
- Log performance with each request
- Standardize the log format for easier parsing later on
- Add blank image for spacing in client side
- Add client side javascript fetch url.
- Add support for background images `url(//some/image/file.png)`