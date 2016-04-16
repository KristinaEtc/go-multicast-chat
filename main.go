package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	name      string
	chsender  chan int
	chreciver chan int
	userNames []string
)

type connStr struct {
	localAddress *net.UDPAddr
	mcastAddress *net.UDPAddr
	mcastConn    *net.UDPConn
	localConn    *net.UDPConn
}

const (
	commGetNicks  string = "GET_USERNAME"
	commMsg       string = "MSG"
	commMyNick    string = "MY_NICK"
	commNickExist string = "NICK_EXIST"
	//commLeave     string = "LEAVED"
	// commChangeNick string = "/NICK"
	commPrivate string = "PRIVATE"
)

const (
	usagePrivate    string = "'/private' command usage: " + userCommPrivate + " NICK MESSAGE"
	usageChangeNick string = "To change nick type '" + userCommChangeNick + " NEW_NICKNAME'"
)

const (
	tagNewName string = "*new"
)

const (
	userCommChangeNick string = "/nick"
	userCommPrivate    string = "/private"
)

//func getMyIP() (addr [4]byte, err error) {
func getMyIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		//os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		//os.Exit(1)
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//return os.Stdout.WriteString(ipnet.IP.String() + "\n")

				//copy(addr[:], ipnet.IP.To4())
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}

var connection = connStr{}
var myRand *rand.Rand

func usage() {
	fmt.Println("****************************************")
	fmt.Println(usagePrivate)
	fmt.Println(usageChangeNick)
	fmt.Println("****************************************")
}

func main() {

	s1 := rand.NewSource(time.Now().UnixNano())
	myRand = rand.New(s1)

	fmt.Printf("nickname？")
	fmt.Scanln(&name)

	myIP, err := getMyIP()

	connection = connStr{}

	connection.localAddress, err = net.ResolveUDPAddr("udp", myIP+":0")
	check(err)
	connection.mcastAddress, err = net.ResolveUDPAddr("udp", "224.0.1.60:8765")
	check(err)

	connection.mcastConn, err = net.ListenMulticastUDP("udp", nil, connection.mcastAddress)
	check(err)
	connection.localConn, err = net.ListenUDP("udp", connection.localAddress)
	check(err)

	message := fmt.Sprintf("%s:%s:%s", commMyNick, tagNewName, name)
	buffer := make([]byte, len(message))
	copy(buffer, []byte(message))
	_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
	check(err)

	go sender(chsender, connection.localConn, connection.mcastAddress)
	go receiver(chreciver, connection.mcastConn)
	<-chsender
	<-chreciver
}

func sender(ch chan int, conn *net.UDPConn, addr *net.UDPAddr) {

	//распределенный консенсунс
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := fmt.Sprintf("%s", scanner.Text())
		fmt.Print("\r")

		var msgParted []string
		msgParted = strings.Split(msg, " ")
		command := msgParted[0]
		switch command {
		case userCommChangeNick:
			{
				if len(msgParted) < 2 || msgParted[1] == name {
					continue
				}
				msg = commMyNick + ":" + name + ":" + msgParted[1]
				name = msgParted[1]

			}
		case userCommPrivate:
			{
				if len(msgParted) < 3 {
					//fmt.Printf("\rSYSTEM: '/private' command usage: %s NICK MESSAGE\n", userCommPrivate)
					fmt.Printf("\r%s\n", usagePrivate)
					fmt.Print("<- ")
					continue
				}
				rowMsg := msg[(len(commPrivate) + len(msgParted[1]) + 3):]
				msg = commPrivate + ":" + msgParted[1] + ":" + name + ": " + rowMsg
				//fmt.Println("row private mes:",rowMsg, "/len: ", len(rowMsg))
				//fmt.Println(msg)
			}
		default:
			{
				msg = commMsg + ":" + name + ":" + msg
				//fmt.Printf("%s\n", msg)
			}
		}

		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)
		fmt.Print("<- ")
	}
	fmt.Print("quit")
	ch <- 1
}

func receiver(ch chan int, conn *net.UDPConn) {
	for {
		b := make([]byte, 256)
		n, addr, err := conn.ReadFromUDP(b)
		check(err)
		//fmt.Printf("n:%d", n)
		rawMsg := string(b[:n])

		//fmt.Println("addr ", addr)

		var msg []string
		msg = strings.SplitN(rawMsg, ":", 3)
		if len(msg) < 1 {
			continue
		}

		//if msg[0] != commMsg && msg[0] != commMyNick {
		//}

		switch msg[0] { //check command type
		case commMsg:
			{
				if msg[1] == name {
					break
				}
				fmt.Printf("\r-> %s: %s\n", msg[1], msg[2])
				fmt.Print("<- ")
				//break
			}
		case commMyNick:
			{
				//fmt.Println("\rrawMsg ", rawMsg)
				fmt.Print("<- ")

				//fmt.Printf("- %d %v %v %v %v\n", i, len(msg[1]), len(name), msg[1] == name, i != 0)

				var who string
				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if i != 0 {
					who = msg[2]
					if msg[2] == name { //names from different ip adds are equal!
						//fmt.Println("i have a nick and he is mine")

						message := fmt.Sprintf("%s:%s", commNickExist, name)
						//s fmt.Println(messa)
						buffer := make([]byte, len(message))
						copy(buffer, []byte(message))

						//_, err = connection.localConn.WriteToUDP(buffer, addr)
						_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
						check(err)
						//break
					}
				} else {
					who = "You"
				}

				if msg[1] == tagNewName {
					fmt.Printf("\r*** %s has joined to chat ***\n", who)
					if i == 0 {
						usage()
					}
				} else {
					if i == 0 {
						fmt.Printf("\r*** %s changed name to %s ***\n", who, msg[2])
					} else {
						fmt.Printf("\r*** %s changed name to %s ***\n", msg[1], msg[2])
					}
				}

				fmt.Print("\r<- ")
				//break
			}
		case commNickExist:
			{

				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if msg[1] == name && (i != 0) { //nick is the same, ip addr is not
					//fmt.Print("commLineExists\n")

					newName := "User" + strconv.Itoa(myRand.Intn(1000))
					fmt.Printf("\rSYSTEM: Nick %s already exists. Changing to %s\n", name, newName)
					fmt.Printf("SYSTEM: %s\n", usageChangeNick)
					fmt.Print("<- ")

					//check(err)

					message := fmt.Sprintf("%s:%s:%s", commMyNick, name, newName)
					name = newName
					//fmt.Println("ip ", message)
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))
					_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
					check(err)

					//break
				}

			}
		case commPrivate:
			{
				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if i != 0 {
					if len(msg) > 2 && msg[1] == name {
						rawMsg = rawMsg[len(msg[0])+len(msg[1])+2:]
						fmt.Printf("\r->[%s] %s\n", msg[0], rawMsg)
						fmt.Print("<- ")
					}
				}
			}
		default:
			{
				fmt.Println("you get strange message; ignore")
				//continue
			}
		}

	}
	ch <- 1
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
