# sslkeylogmerge
An application to merge multiple SSL Keylog Files into one

Suppose you want to inspect TLS traffic from multiple applications at once using Wireshark, and each of these applications supports the SSLKEYLOGFILE environment variable to dump their connection keys to a file.

Unfortunately, Wireshark only supports reading secrets from one SSLKEYLOGFILE at a time.

This application will read each application's separate SSLKEYLOGFILE and combine them into a single file for Wireshark to consume.

## Installation
```shell
go build . -o sslkeylogmerge
```

## Usage
```text
USAGE:
   sslkeylogmerge [global options] command [command options] 

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --output file, -o file                                               output file [$SSLKEYLOGFILE]
   --input file, -i file [ --input file, -i file ]                      individual input file(s)
   --watch directory, -w directory [ --watch directory, -w directory ]  watch directory(ies)
   --help, -h                                                           show help
```

## Examples
### Merging the SSL key logs from cURL, Chrome, and Firefox
1) Start the merge application:
```shell
./sslkeylogmerge -o ~/sslkeys.log \
    -i ~/sslkeylogs/curl.log \
    -i ~/sslkeylogs/chrome.log \
    -i ~/sslkeylogs/firefox.log 
```

2) Open Firefox 
```shell
SSLKEYLOGFILE=~/sslkeylogs/firefox.log open -a firefox
```

3) Open Chrome
```shell
SSLKEYLOGFILE=~/sslkeylogs/chrome.log open -a chrome
```

4) Run your cURL command
```shell
SSLKEYLOGFILE=~/sslkeylogs/curl.log curl https://example.net
```

5) Configure Wireshark to read TLS secrets from ~/sslkeys.log

### Merging by watching a directory
1) Start the merge application:
```shell
./sslkeylogmerge -o ~/sslkeys.log \
    -w ~/sslkeylogs/ 
```

2) Continue from step 2 in the first example

