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

var layout string = "2006-01-02 15:04:05 -0700 MST"

//for generation username
var s rand.Source = rand.NewSource(time.Now().UnixNano())
var myRand = rand.New(s)

func (g *globalData) checkPing() {
	for {

		timer := time.NewTimer(time.Second * 5)
		<-timer.C

		//sending ping to others
		msg := commPing + "|" + g.user.name
		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
		check(err)

		//checking ping from others
		for user, lastPing := range g.lastUserPing {
			diff := time.Now().Sub(lastPing)
			if diff.Seconds() > 10 {
				fmt.Printf("\r*** %s leaved the chat  ***\n", user)
				delete(g.lastUserPing, user)
				fmt.Print("<- ")
			}
		}
	}
}

func (g *globalData) checkMsgStatus() {

	for {
		timer := time.NewTimer(time.Second * 10)
		<-timer.C

		//checking time were we sent message
		for wasSended, msgStat := range g.sendedMsg {

			//msgStat = *msgStat
			fmt.Println("kv:", wasSended, " ", msgStat)

			diff := time.Now().Sub(wasSended)
			if diff.Seconds() > 10 {

				//if all users got a message - delete it from the store
				if msgStat.userCount == len(msgStat.answerStatus) {
					delete(g.sendedMsg, wasSended)
				} else {
					fmt.Println(len(g.lastUserPing), len(msgStat.answerStatus), "\n")
					//searching who have not received
					for user, _ := range g.lastUserPing {
						if _, ok := (msgStat.answerStatus)[user]; !ok {
							//delete(g.sendedMsg, wasSended)
							/*buffer := make([]byte, len(msgStat.msgBody))
							copy(buffer, []byte(msgStat.msgBody))
							_, err := g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
							check(err)*/
						}
					}
				}
			}
		}
	}
}

func (g *globalData) getMsgStore(id time.Time) *msgStore {
	store, ok := g.sendedMsg[id]
	if !ok {
		answerStatus := make(map[string]bool)
		store = &msgStore{answerStatus: answerStatus}
		g.sendedMsg[id] = store
	}
	return store
}

func (g *globalData) sender() {

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := fmt.Sprintf("%s", scanner.Text())
		fmt.Print("\r")

		var msgParted []string = strings.Split(msg, " ")
		command := msgParted[0]
		switch command {
		case userCommChangeNick:
			{
				if len(msgParted) < 2 || msgParted[1] == g.user.name || strings.Contains(msgParted[1], "/") {
					fmt.Printf("\r*** Wrong command usage: %s ***", usageChangeNick)
					fmt.Print("\r")
					continue
				}
				msg = commMyNick + "|" + g.user.name + "|" + msgParted[1]
				delete(g.lastUserPing, g.user.name)
				g.user.name = msgParted[1]

			}
		case userCommPrivate:
			{
				if len(msgParted) < 3 {
					fmt.Printf("\rWrong command usage: %s\n", usagePrivate)
					fmt.Print("<- ")
					continue
				}

				var userExist = false
				for user, _ := range g.lastUserPing {
					if user == msgParted[1] {
						userExist = true
						break
					}
				}
				if userExist == false {
					fmt.Printf("\rNo user with such name; ignored\n")
					fmt.Print("<- ")
					continue
				}
				rawMsg := msg[(len(commPrivate) + len(msgParted[1]) + 3):]
				msg = commPrivate + "|" + msgParted[1] + "|" + g.user.name + "| " + rawMsg
			}
		case userCommExit:
			{
				msg = commExit + "|" + g.user.name
				timeID := time.Now()
				msg = timeID.String() + "|" + msg
				fmt.Println("*** Bye ***")
				buffer := make([]byte, len(msg))
				copy(buffer, []byte(msg))
				_, err := g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
				check(err)
				os.Exit(0)
			}
		case userCommGetUsers:
			{
				fmt.Println("\rUsers:")
				for user, lastPing := range g.lastUserPing {
					diff := time.Now().Sub(lastPing)
					if diff.Seconds() < 5 {
						fmt.Print(user, "\t")
					} else {
						fmt.Println(diff.Seconds(), "/you sholdnt see this message")
						delete(g.lastUserPing, user)
					}
				}
				fmt.Print("\n<- ")
				continue
			}
		default: //just message
			{
				if len(msg) != 0 && msg[:1] == "/" {
					fmt.Println("\rCommand not found")
					fmt.Print("<- ")
					continue
				}
				msg = commMsg + "|" + g.user.name + "|" + msg
			}
		}

		timeID := time.Now()
		msg = timeID.String() + "|" + msg
		g.getMsgStore(timeID).msgBody = msg
		g.getMsgStore(timeID).userCount = len(g.lastUserPing)

		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
		check(err)
		fmt.Print("<- ")
	}
	os.Exit(0)
}

func (g *globalData) receiver() {

	for {
		//reading message
		b := make([]byte, 256)
		n, addr, err := g.conn.mcastConn.ReadFromUDP(b)
		check(err)
		rawMsg := string(b[:n])

		//if rawMsg == "" {
		//	continue
		//}

		//parsing msg
		var msg []string
		msg = strings.SplitN(rawMsg, "|", 4)

		//system command (ping or got_msg answer)
		switch msg[0] {
		case commPing:
			{
				g.lastUserPing[msg[1]] = time.Now()
				continue
			}
		case commGotMsg:
			{
				//fmt.Printf("\rgot commGotMsg: %s\n", rawMsg)
				timeID, err := time.Parse(layout, msg[1])
				check(err)

				g.getMsgStore(timeID).answerStatus[msg[2]] = true
				delete(g.sendedMsg, timeID)
				//	}

				continue
			}
		}

		//fmt.Printf("\rrawMsg-%s-\n", rawMsg)
		//fmt.Print("<- ")

		msgTimeID, err := time.Parse(layout, msg[0])
		check(err)

		i := strings.Compare(addr.String(), g.conn.localConn.LocalAddr().String())
		var fromWho string
		if i == 0 {
			fromWho = "You"
			g.getMsgStore(msgTimeID).answerStatus[g.user.name] = true
		} else { //sending GOT_MESSAGE command

			fromWho = "fromOthers"
			message := fmt.Sprintf("%s|%s|%s", commGotMsg, msg[0], g.user.name)
			//fmt.Printf("\rsending: %s-\n", message)
			//fmt.Print("<- ")

			buffer := make([]byte, len(message))
			copy(buffer, []byte(message))
			_, err = g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
			check(err)
		}

		switch msg[1] { //check command type
		case commMsg:
			{
				if msg[2] == g.user.name {
					break
				}
				fmt.Printf("\r-> %s: %s\n", msg[2], msg[3])
				fmt.Print("<- ")

			}
		case commMyNick:
			{
				fmt.Print("<- ")

				if fromWho == "fromOthers" {

					if msg[2] != tagNewName {
						fromWho = msg[2]
					} else {
						fromWho = msg[3]
					}

					if msg[3] == g.user.name { //names from different ip adds are equal!
						timeID := time.Now()

						message := fmt.Sprintf("%s|%s|%s", timeID.String(), commNickExist, g.user.name)
						buffer := make([]byte, len(message))
						copy(buffer, []byte(message))
						_, err = g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
						check(err)

						g.getMsgStore(msgTimeID).msgBody = message

					} else { //nick is ok, adding it to userNicks
						g.lastUserPing[msg[3]] = time.Now()
					}
					delete(g.lastUserPing, msg[2])
				}

				if msg[2] == tagNewName {
					fmt.Printf("\r*** %s has joined to chat ***\n", fromWho)
					if fromWho == "You" {
						g.lastUserPing[g.user.name] = time.Now()
						usage()
					}
				} else {
					fmt.Printf("\r*** %s changed name to %s ***\n", fromWho, msg[3])
				}
				fmt.Print("\r<- ")
			}
		case commNickExist:
			{
				if msg[2] == g.user.name && fromWho != "You" { //nick is the same, ip addr is not

					delete(g.lastUserPing, g.user.name)
					newName := "User" + strconv.Itoa(myRand.Intn(1000))
					fmt.Printf("\rSYSTEM: Nick %s already exists. Changing to %s\n", g.user.name, newName)
					fmt.Printf("SYSTEM: %s\n", usageChangeNick)
					fmt.Print("<- ")

					timeId := time.Now()
					timeId, err = time.Parse(layout, timeId.String())
					check(err)

					message := fmt.Sprintf("%s|%s|%s|%s", timeId.String(), commMyNick, g.user.name, newName)

					g.user.name = newName
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))
					_, err = g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
					check(err)

					g.getMsgStore(msgTimeID).msgBody = message
				}
			}
		case commPrivate:
			{
				if fromWho == "fromOthers" {
					if len(msg) > 3 && msg[2] == g.user.name {
						rawMsg = rawMsg[len(msg[0])+len(msg[1])+len(msg[2])+3:]
						fmt.Printf("\r->[%s] %s\n", msg[1], rawMsg)
						fmt.Print("<- ")
					}
				}
			}
		case commExit:
			{
				fmt.Printf("\r*** %s leaved the chat  ***\n", msg[2])
				delete(g.lastUserPing, msg[2])
				fmt.Print("<- ")
			}

		default:
			{
				fmt.Printf("you've got a strange message; ignore %s", rawMsg)
			}

		}
	}
}

func main() {
	var userName string

	fmt.Printf("nickname？")
	fmt.Scanln(&userName)
	//New(name)

	global := globalData{}
	global.user = userInfo{name: userName}
	global.lastUserPing = make(map[string]time.Time)
	global.sendedMsg = make(map[time.Time]*msgStore)

	//connection settings
	myIP, err := getMyIP()
	check(err)

	conn := connPointers{}
	conn.localAddress, err = net.ResolveUDPAddr("udp", myIP+":0")
	check(err)
	conn.mcastAddress, err = net.ResolveUDPAddr("udp", "224.0.1.60:8765")
	check(err)
	conn.mcastConn, err = net.ListenMulticastUDP("udp", nil, conn.mcastAddress)
	check(err)
	conn.localConn, err = net.ListenUDP("udp", conn.localAddress)
	check(err)

	global.conn = conn

	//we have a new user: sending it to other users
	timeId := time.Now()
	timeId, err = time.Parse(layout, timeId.String())
	//check(err)
	//fmt.Printf("/%s/\n", timeId.String())

	message := fmt.Sprintf("%s|%s|%s|%s", timeId.String(), commMyNick, tagNewName, userName)
	buffer := make([]byte, len(message))
	copy(buffer, []byte(message))
	_, err = global.conn.localConn.WriteToUDP(buffer, global.conn.mcastAddress)
	check(err)

	//adding a new user to userlist
	global.lastUserPing[userName] = timeId
	//msgStat := msgStore{msgBody: message}
	//global.sendedMsg[timeId] = msgStat

	//starting goroutines, that will be waiting new messages
	go global.sender()
	go global.receiver()
	go global.checkPing()
	go global.checkMsgStatus()
	<-global.user.chsender
	<-global.user.chreciver
}
