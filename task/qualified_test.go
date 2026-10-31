package task
import("net";"testing";"github.com/XIU2/CloudflareSpeedTest/utils")
func TestFinalTargetReplenishesDownloadFailures(t *testing.T){
 ips:=[]*net.IPAddr{{IP:net.ParseIP("192.0.2.1")},{IP:net.ParseIP("192.0.2.2")},{IP:net.ParseIP("192.0.2.3")},{IP:net.ParseIP("192.0.2.4")}}
 calls:=0
 probe:=func(ip *net.IPAddr)*utils.PingData{return &utils.PingData{IP:ip}}
 verify:=func(ip *net.IPAddr)(float64,bool,bool){calls++;return 100,ip.String()!="192.0.2.1",false}
 got:=selectQualified(ips,2,probe,verify)
 if len(got)!=2 || calls!=3 || got[0].IP.String()!="192.0.2.2" {t.Fatal(len(got),calls)}
 calls=0;got=selectQualified(ips,5,probe,verify);if len(got)!=3||calls!=4{t.Fatal(len(got),calls)}
 calls=0;got=selectQualified(ips,2,probe,func(*net.IPAddr)(float64,bool,bool){calls++;return 0,false,true});if len(got)!=0||calls!=1{t.Fatal(len(got),calls)}
}
