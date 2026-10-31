package task

import (
 "bufio"
 "encoding/binary"
 "fmt"
 "log"
 "math/rand"
 "net"
 "os"
 "sort"
 "strings"
)

var SampleCount int
type interval struct { lo, hi uint64 }

// Sample without enumerating CIDRs. Merge overlaps, then use a sparse partial
// Fisher-Yates shuffle over the union so every IPv4 has equal probability.
func sampleIPv4(lines []string, n int) ([]*net.IPAddr, error) {
 ranges := []interval{}
 for _, line := range lines {
  line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
  if line == "" { continue }
  if !strings.Contains(line, "/") { line += "/32" }
  ip, network, err := net.ParseCIDR(line)
  if err != nil || ip.To4() == nil { return nil, fmt.Errorf("-sample needs valid IPv4 CIDR: %s", line) }
  ones, bits := network.Mask.Size()
  if bits != 32 { return nil, fmt.Errorf("IPv6 sampling is not supported") }
  lo := uint64(binary.BigEndian.Uint32(network.IP.To4()))
  ranges = append(ranges, interval{lo, lo+(uint64(1)<<uint(32-ones))-1})
 }
 sort.Slice(ranges, func(i,j int) bool { return ranges[i].lo < ranges[j].lo })
 merged := []interval{}
 for _, r := range ranges {
  if len(merged)>0 && r.lo <= merged[len(merged)-1].hi+1 {
   if r.hi>merged[len(merged)-1].hi { merged[len(merged)-1].hi=r.hi }
  } else { merged=append(merged,r) }
 }
 cumulative:=make([]uint64,len(merged)); total:=uint64(0)
 for i,r:=range merged { total+=r.hi-r.lo+1; cumulative[i]=total }
 if n<0 || uint64(n)>total { return nil,fmt.Errorf("sample %d exceeds unique address count %d",n,total) }
 swaps:=make(map[uint64]uint64,n)
 lookup:=func(k uint64)uint64 { if v,ok:=swaps[k];ok{return v};return k }
 result:=make([]*net.IPAddr,0,n)
 for i:=0;i<n;i++ {
  remaining:=total-uint64(i)
  index:=uint64(rand.Int63n(int64(remaining)))
  offset:=lookup(index)
  swaps[index]=lookup(remaining-1)
  delete(swaps,remaining-1)
  j:=sort.Search(len(cumulative),func(j int)bool{return cumulative[j]>offset})
  before:=uint64(0);if j>0{before=cumulative[j-1]}
  value:=merged[j].lo+offset-before
  ip:=make(net.IP,4);binary.BigEndian.PutUint32(ip,uint32(value))
  result=append(result,&net.IPAddr{IP:ip})
 }
 return result,nil
}
func loadSampledIPv4() []*net.IPAddr {
 var lines []string
 if IPText!="" { lines=strings.Split(IPText,",") } else {
  name:=IPFile;if name==""{name=defaultInputFile}
  f,err:=os.Open(name);if err!=nil{log.Fatal(err)};defer f.Close()
  scan:=bufio.NewScanner(f)
  for scan.Scan(){lines=append(lines,scan.Text())}
  if err:=scan.Err();err!=nil{log.Fatal(err)}
 }
 ips,err:=sampleIPv4(lines,SampleCount);if err!=nil{log.Fatal(err)}
 return ips
}
