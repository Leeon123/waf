# waf
Simple golang tcp reverse proxy with filter

Since added the limitation of connection per ip,

it could easily block the non-proxies tcp/http flood.

Proxied tcp/http flood need some time to block.

## Function
- **Anti-cc**
  - Limit the connections per ip
  - Limit the requests per connection
  - Limit the requests per second of every ip
- **Block IP system**
  - Auto block ip trigger the limitation
  - ~~Unblock all ip every 30 second(might be change)~~
  - Unban the blocked ip until you want
- **Check validity of request**
  - Unfinished
- **Block injection**
  - Unfinished
- **Filter the sensitive url**
  - Unfinished
  
## Usage
You can change the setting below:
```
	// You can edit this
	waf_port                 = "0.0.0.0:80"     //your waf address
	real_port                = "localhost:1337" //your real address
	pps_per_ip_limit         = 10               //Limit the packets per second of ip
	connection_limit         = 10               //Limit the connections of ip
	banned_time      float64 = 60               //Blocking time of the banned ip
```

## TODO
- [x] Anti-cc
- [x] Block IP system
- [ ] Check validity of request
- [ ] Block injection 
- [ ] Filter the sensitive url

## Experiment

Tested with 1400+ socks4 proxies, it takes some time to block all the ips.

**Attack side** (4c8g) using socks4 cc

![](https://i.imgur.com/Ew5veBq.png)

**Server side** (2c2g) using apache2 http server

![](https://i.imgur.com/zR6fd3b.png)

