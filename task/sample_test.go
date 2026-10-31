package task
import("testing";"net")
func TestSampleUniqueAndBounds(t *testing.T){
 ips,e:=sampleIPv4([]string{"192.0.2.0/24","192.0.2.0/25","192.0.2.10"},256)
 if e!=nil{t.Fatal(e)};seen:=map[string]bool{}
 _,n,_:=net.ParseCIDR("192.0.2.0/24")
 for _,ip:=range ips{if seen[ip.String()]||!n.Contains(ip.IP){t.Fatal(ip)};seen[ip.String()]=true}
 if len(seen)!=256{t.Fatal(len(seen))}
 if _,e:=sampleIPv4([]string{"192.0.2.0/30"},5);e==nil{t.Fatal("oversampling accepted")}
}
func TestSampleEntireIPv4SpaceBounded(t *testing.T){
 ips,e:=sampleIPv4([]string{"0.0.0.0/0"},400);if e!=nil||len(ips)!=400{t.Fatal(e,len(ips))}
}
func TestSampleSmallAndInvalid(t *testing.T){
 ips,e:=sampleIPv4([]string{"# comment","192.0.2.1/32","192.0.2.2"},2)
 if e!=nil||len(ips)!=2{t.Fatal(e)}
 for _,s:=range []string{"bad","2001:db8::/32"}{if _,e:=sampleIPv4([]string{s},1);e==nil{t.Fatal(s)}}
}
