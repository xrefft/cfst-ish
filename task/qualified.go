package task

import (
 "fmt"
 "io"
 "net"
 "net/http"
 "time"

 "github.com/XIU2/CloudflareSpeedTest/utils"
)

var FinalTarget int
var VerifyBytes int64 = 1048576
var VerifyRounds = 1
var VerifyTimeout = 30*time.Second
var VerifyRatio = 0.5
var VerifyBudget int64 = 300*1048576

// Select only fully qualified candidates; failed downloads do not consume target slots.
func selectQualified(ips []*net.IPAddr, target int, probe func(*net.IPAddr) *utils.PingData, verify func(*net.IPAddr) (float64, bool, bool)) utils.DownloadSpeedSet {
 result := make(utils.DownloadSpeedSet,0)
 for i,ip := range ips {
  fmt.Printf("[%d/%d] %s qualified=%d/%d\n",i+1,len(ips),ip,len(result),target)
  data:=probe(ip)
  if data==nil {continue}
  speed,ok,stop:=verify(ip)
  if ok {result=append(result,utils.CloudflareIPData{PingData:data,DownloadSpeed:speed}); fmt.Printf("PASS %s qualified=%d/%d\n",ip,len(result),target)}
  if stop || len(result)>=target {break}
 }
 return result
}

func RunQualified() utils.DownloadSpeedSet {
 checkPingDefault()
 ips:=loadIPRanges()
 p:=&Ping{}
 remaining:=VerifyBudget
 result:=selectQualified(ips,FinalTarget,func(ip *net.IPAddr)*utils.PingData{
  recv,delay,colo:=p.checkConnection(ip)
  if recv!=PingTimes{return nil}
  data:=&utils.PingData{IP:ip,Sended:PingTimes,Received:recv,Delay:delay/time.Duration(recv),Colo:colo}
  if data.Delay<utils.InputMinDelay || data.Delay>utils.InputMaxDelay{return nil}
  return data
 },func(ip *net.IPAddr)(float64,bool,bool){
  min,max:=0.0,0.0
  for r:=0;r<VerifyRounds;r++ {
   if remaining<VerifyBytes {fmt.Println("download byte budget exhausted");return 0,false,true}
   speed,n,err:=verifyDownload(ip)
   remaining-=n
   if err!=nil || speed<MinSpeed*1048576 {fmt.Printf("download rejected: bytes=%d speed=%.3f MiB/s error=%v\n",n,speed/1048576,err);return 0,false,false}
   if r==0 || speed<min {min=speed};if speed>max{max=speed}
  }
  return min,max>0 && min/max>=VerifyRatio,false
 })
 fmt.Printf("Finished: %d/%d qualified IPs; candidate limit=%d; body bytes used=%d\n",len(result),FinalTarget,len(ips),VerifyBudget-remaining)
 return result
}

// Download exactly VerifyBytes with verified TLS, directly to the candidate IP.
// A short body, HTTP error, redirect or timeout is never a passing download.
func verifyDownload(ip *net.IPAddr)(float64,int64,error){
 url:=URL
 if url==defaultURL {url=fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d",VerifyBytes)}
 transport:=&http.Transport{DialContext:getDialContext(ip),DisableKeepAlives:true,DisableCompression:true}
 defer transport.CloseIdleConnections()
 client:=&http.Client{Transport:transport,Timeout:VerifyTimeout,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 return verifyTransfer(client,url)
}

func verifyTransfer(client *http.Client,url string)(float64,int64,error){
 req,err:=http.NewRequest("GET",url,nil);if err!=nil{return 0,0,err}
 if req.URL.Scheme!="https" {return 0,0,fmt.Errorf("verification requires HTTPS")}
 req.Header.Set("Range",fmt.Sprintf("bytes=0-%d",VerifyBytes-1))
 start:=time.Now()
 response,err:=client.Do(req);if err!=nil{return 0,0,err}
 defer response.Body.Close()
 if response.StatusCode!=200 && response.StatusCode!=206{return 0,0,fmt.Errorf("HTTP %d",response.StatusCode)}
 n,err:=io.CopyN(io.Discard,response.Body,VerifyBytes)
 return float64(n)/time.Since(start).Seconds(),n,err
}
