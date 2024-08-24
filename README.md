# EasyDNS

A very very simple DNS server for development/testing purposes. (For example if you need to mock something)

## Build 

```bash
# Will build on linux, windows and Mac (Apple chip)
bash build.sh
```

# Get config template

Locate your binary under the build folder and run:

```bash
./easydns config -save -config-path /path/to/config.json
```

Edit the configuration file and then start the server:

```bash
./easydns run -config-path /path/to/config.json
```


That's it. As said it's very simple.
