package utils
import("testing";"time")
func TestDelayFilterDoesNotDropLaterLossGroups(t *testing.T){
 old:=InputMaxDelay;defer func(){InputMaxDelay=old}();InputMaxDelay=100*time.Millisecond
 data:=PingDelaySet{{PingData:&PingData{Sended:4,Received:4,Delay:200*time.Millisecond}}, {PingData:&PingData{Sended:4,Received:3,Delay:50*time.Millisecond}}}
 if got:=data.FilterDelay();len(got)!=1||got[0].Received!=3{t.Fatal(got)}
}
