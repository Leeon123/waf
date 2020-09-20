# waf
Simple golang tcp reverse proxy with filter

Since added the limitation of connection per ip,

it could easily block the non-proxies tcp/http flood.

## Function
- **Anti-cc**
  - Limit the connections per ip
  - Limit the requests per connection
  - Limit the requests per second of every ip
- **Block IP system**
  - Auto block ip trigger the limitation
  - Unblock all ip every 30 second(might be change)
- **Check validity of request**
  - Unfinished
- **Block injection**
  - Unfinished
- **Filter the sensitive url**
  - Unfinished

## TODO
- [x] Anti-cc
- [x] Block IP system
- [ ] Check validity of request
- [ ] Block injection 
- [ ] Filter the sensitive url

## Experiment
**Attack side** (4c8g) using socks4 cc

![](https://i.imgur.com/Ew5veBq.png)

**Server side** (2c2g) using apache2 http server

![](https://i.imgur.com/pvnFLB7.png)
