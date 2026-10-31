package task

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/XIU2/CloudflareSpeedTest/utils"
)

const (
	maxRoutine        = 1000
	defaultRoutines   = 200
	defaultPort       = 443
	defaultPingTimes  = 4
)

var (
	ISH bool
	StopAfter int
	ConnectTimeout = time.Second
	Routines      = defaultRoutines
	TCPPort   int = defaultPort
	PingTimes int = defaultPingTimes
)

type Ping struct {
	wg      *sync.WaitGroup
	m       *sync.Mutex
	ips     []*net.IPAddr
	csv     utils.PingDelaySet
	control chan bool
	bar     *utils.Bar
}

func checkPingDefault() {
	if Routines <= 0 {
		Routines = defaultRoutines
	}
	if Routines > maxRoutine { Routines = maxRoutine }
	if TCPPort <= 0 || TCPPort > 65535 {
		TCPPort = defaultPort
	}
	if PingTimes <= 0 {
		PingTimes = defaultPingTimes
	}
}

func NewPing() *Ping {
	checkPingDefault()
	ips := loadIPRanges()
	var bar *utils.Bar
	if !ISH && StopAfter == 0 { bar = utils.NewBar(len(ips), "可用:", "") }
	return &Ping{
		wg:      &sync.WaitGroup{},
		m:       &sync.Mutex{},
		ips:     ips,
		csv:     make(utils.PingDelaySet, 0),
		control: make(chan bool, Routines),
		bar:     bar,
	}
}

func (p *Ping) Run() utils.PingDelaySet {
	if len(p.ips) == 0 {
		return p.csv
	}
	if Httping {
		utils.Cyan.Printf("开始延迟测速（模式：HTTP, 端口：%d, 范围：%v ~ %v ms, 丢包：%.2f)\n", TCPPort, utils.InputMinDelay.Milliseconds(), utils.InputMaxDelay.Milliseconds(), utils.InputMaxLossRate)
	} else {
		utils.Cyan.Printf("开始延迟测速（模式：TCP, 端口：%d, 范围：%v ~ %v ms, 丢包：%.2f)\n", TCPPort, utils.InputMinDelay.Milliseconds(), utils.InputMaxDelay.Milliseconds(), utils.InputMaxLossRate)
	}
	if ISH || StopAfter > 0 {
		qualified := 0
		for i, ip := range p.ips {
			fmt.Printf("[%d/%d] %s ...\n", i+1, len(p.ips), ip.String())
			if p.tcpingHandler(ip) { qualified++ }
			fmt.Printf("completed=%d qualifying=%d\n", i+1, qualified)
			if StopAfter > 0 && qualified >= StopAfter { break }
		}
		sort.Sort(p.csv)
		return p.csv
	}
	for _, ip := range p.ips {
		p.wg.Add(1)
		p.control <- false
		go p.start(ip)
	}
	p.wg.Wait()
	p.bar.Done()
	sort.Sort(p.csv)
	return p.csv
}

func (p *Ping) start(ip *net.IPAddr) {
	defer p.wg.Done()
	p.tcpingHandler(ip)
	<-p.control
}

// bool connectionSucceed float32 time
func (p *Ping) tcping(ip *net.IPAddr) (bool, time.Duration) {
	startTime := time.Now()
	var fullAddress string
	if isIPv4(ip.String()) {
		fullAddress = fmt.Sprintf("%s:%d", ip.String(), TCPPort)
	} else {
		fullAddress = fmt.Sprintf("[%s]:%d", ip.String(), TCPPort)
	}
	conn, err := net.DialTimeout("tcp", fullAddress, ConnectTimeout)
	if err != nil {
		return false, 0
	}
	defer conn.Close()
	duration := time.Since(startTime)
	return true, duration
}

// pingReceived pingTotalTime
func (p *Ping) checkConnection(ip *net.IPAddr) (recv int, totalDelay time.Duration, colo string) {
	if Httping {
		recv, totalDelay, colo = p.httping(ip)
		return
	}
	colo = "" // TCPing 不获取 colo
	for i := 0; i < PingTimes; i++ {
		if ok, delay := p.tcping(ip); ok {
			recv++
			totalDelay += delay
		}
	}
	return
}

func (p *Ping) appendIPData(data *utils.PingData) {
	p.m.Lock()
	defer p.m.Unlock()
	p.csv = append(p.csv, utils.CloudflareIPData{
		PingData: data,
	})
}

// handle tcping
func (p *Ping) tcpingHandler(ip *net.IPAddr) bool {
	recv, totalDelay, colo := p.checkConnection(ip)
	var data *utils.PingData
	if recv > 0 {
		data = &utils.PingData{IP: ip, Sended: PingTimes, Received: recv,
			Delay: totalDelay / time.Duration(recv), Colo: colo}
	}
	// Read and mutate the result slice under the same lock. Upstream read len
	// concurrently with append, producing a race even though append was locked.
	p.m.Lock()
	if data != nil { p.csv = append(p.csv, utils.CloudflareIPData{PingData: data}) }
	if p.bar != nil { p.bar.Grow(1, strconv.Itoa(len(p.csv))) }
	p.m.Unlock()
	if data == nil { return false }
	loss := float32(PingTimes-recv) / float32(PingTimes)
	return data.Delay >= utils.InputMinDelay && data.Delay <= utils.InputMaxDelay && loss <= utils.InputMaxLossRate
}
