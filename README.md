# go-wondershaper

Golang port of wondershaper: an utility for limiting an adapter's bandwidth.

Note: All API calls except for `wondershaper.Status()` require elevated permissions for `/sbin/tc`.

## Installation

`go get github.com/mysteriumnetwork/go-wondershaper`

## Example

```go
shaper := wondershaper.New()
shaper.Stdout = os.Stdout
shaper.Stderr = os.Stderr
err := shaper.LimitDownlink("eth0", 1024) // Limits download speed to 1024Kbps
if err != nil {
    log.Fatalln("Could not limit downlink", err)
}
```

## See also

[wondershaper](https://github.com/magnific0/wondershaper) (c) 2002-2017 Bert Hubert ahu@ds9a.nl, Jacco Geul jacco@geul.net, Simon Séhier simon@sehier.fr
