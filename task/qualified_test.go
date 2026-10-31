package task
import("net";"testing";"net/http";"net/http/httptest";"time";"github.com/XIU2/CloudflareSpeedTest/utils")
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

func TestVerifyTransferRejectsShortAndHTTPError(t *testing.T){
 old:=VerifyBytes;defer func(){VerifyBytes=old}();VerifyBytes=16
 for _,tc:=range []struct{status,size int;pass bool}{{200,16,true},{200,4,false},{206,16,true},{500,16,false},{302,16,false}} {
  server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(tc.status);w.Write(make([]byte,tc.size))}))
  client:=server.Client();client.Timeout=time.Second
  speed,n,err:=verifyTransfer(client,server.URL)
  server.Close()
  if tc.pass && (err!=nil||n!=16||speed<=0){t.Fatalf("pass failed: %v %d",err,n)}
  if !tc.pass && err==nil {t.Fatalf("accepted %v",tc)}
 }
 server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){time.Sleep(50*time.Millisecond);w.Write(make([]byte,16))}))
 defer server.Close();client:=server.Client();client.Timeout=10*time.Millisecond
 if _,_,err:=verifyTransfer(client,server.URL);err==nil{t.Fatal("timeout accepted")}
}
