package task
import("net";"strconv";"testing";"time";"sync/atomic";"github.com/XIU2/CloudflareSpeedTest/utils")
func TestStopAndConcurrentProbes(t *testing.T){
 listener,err:=net.Listen("tcp4","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 var accepted int64
 go func(){for{c,e:=listener.Accept();if e!=nil{return};atomic.AddInt64(&accepted,1);c.Close()}}()
 defer listener.Close()
 _,port,_:=net.SplitHostPort(listener.Addr().String());TCPPort,_=strconv.Atoi(port)
 PingTimes=4;ConnectTimeout=time.Second;utils.InputMinDelay=0;utils.InputMaxDelay=10*time.Second;utils.InputMaxLossRate=0
 IPText="127.0.0.1,127.0.0.1,127.0.0.1";SampleCount=0;ISH=true;StopAfter=2
 data:=NewPing().Run();if len(data)!=2{t.Fatalf("early stop returned %d",len(data))}
 for _,d:=range data{if d.Received!=4{t.Fatal(d)}}
 // Exercise the original concurrent scheduler under go test -race.
 ISH=false;StopAfter=0;Routines=2
 data=NewPing().Run();if len(data)!=3{t.Fatal(len(data))}
}
