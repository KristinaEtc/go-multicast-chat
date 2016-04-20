package main

import (
	"bufio"
	"fmt"
	//"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func New(userName string) globalData {

	global := globalData{}
	global.user = userInfo{name: userName}
	global.lastUserPing = make(map[string]time.Time)
	global.recivedMsg = make(map[time.Time]string)

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
	timeID := time.Now()
	message := fmt.Sprintf("%v:%s:%s", timeID, tagNewName, userName)
	buffer := make([]byte, len(message))
	copy(buffer, []byte(message))
	_, err = connection.localConn.WriteToUDP(buffer, global.conn.mcastAddress)
	check(err)

	//adding a new user to userlist
	global.lastUserPing[userName] = timeID

	//starting goroutines, that will be waiting new messages
	go global.sender()
	go global.receiver()
	go global.checkPing()
	go global.checkMsgStatus()
}

func (g *globalData) checkPing(ch chan int, conn *net.UDPConn, addr *net.UDPAddr) {
	for {

		timer := time.NewTimer(time.Second * 5)
		<-timer.C

		//sending ping to others
		msg := commPing + ":" + name
		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
		check(err)

		//checking ping from others
		for user, lastPing := range g.lastUserPing {
			diff := time.Now().Sub(lastPing)
			if diff.Seconds() > 5 && user != g.user {
				fmt.Printf("\r*** %s leaved the chat  ***\n", user)
				delete(g.lastUserPing, user)
				fmt.Print("<- ")
			}
		}
	}
}

func (g *globalData) checkMsgStatus() {

	for {
		timer := time.NewTimer(time.Second * timeToCheckUsers)
		<-timer.C

		//checking time were we sent message
		for wasSended, msgBody := range g.recivedMsg {
			diff := time.Now().Sub(wasSended)
			if diff.Seconds() > 5 {
				fmt.Printf("\r*** Message >%s< was not sended  ***\n", msgBody)
				delete(g.recivedMsg, wasSended)
				fmt.Print("<- ")
				continue
			}
		}
	}
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
				if len(msgParted) < 2 || msgParted[1] == g.user || strings.Contains(msgParted[1], "/") {
					fmt.Printf("\r*** Wrong command usage: %s ***", usageChangeNick)
					fmt.Print("\r")
					continue
				}
				msg = commMyNick + ":" + g.user + ":" + msgParted[1]
				g.user = msgParted[1]
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
				msg = commPrivate + ":" + msgParted[1] + ":" + g.user + ": " + rawMsg
			}
		case userCommExit:
			{
				msg = commExit + ":" + g.name
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
				if msg[:1] == "/" {
					fmt.Println("\rCommand not found")
					fmt.Print("<- ")
					continue
				}
				msg = commMsg + ":" + name + ":" + msg
			}
		}

		timeID := time.Now()
		msg = timeID + ":" + msg

		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)
		fmt.Print("<- ")
	}
	os.Exit(0)
}

func (g *globalData) receiver() {

	for {
		//reading message
		b := make([]byte, 256)
		n, addr, err := conn.ReadFromUDP(b)
		check(err)
		rawMsg := string(b[:n])

		//parsing msg
		var msg []string
		msg = strings.SplitN(rawMsg, ":", 4)
		if len(msg) < 2 {
			continue
		}

		if g.recivedMsg[msg[0]] != nil {
			fmt.Printf("\rYou've got message twice; ignore\n")
			fmt.Print("<- ")
			continue
		} else {
			timeID := time.Now()
			message := fmt.Sprintf("%s:%s:%s", timeID, commGotMsg, msg[0])
			g.recivedMsg[timeID] = message

			buffer := make([]byte, len(message))
			copy(buffer, []byte(message))
			_, err = g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
			check(err)
		}

		switch msg[1] { //check command type
		case commMsg:
			{
				if msg[2] == g.user {
					break
				}
				fmt.Printf("\r-> %s: %s\n", msg[2], msg[3])
				fmt.Print("<- ")
			}
		case commMyNick:
			{
				fmt.Print("<- ")

				var who string
				i := strings.Compare(addr.String(), g.conn.localConn.LocalAddr().String())
				if i != 0 {
					who = msg[3]
					if msg[3] == g.user { //names from different ip adds are equal!
						timeID := time.Now()

						message := fmt.Sprintf("%s:%s:%s", timeID, commNickExist, g.user)
						buffer := make([]byte, len(message))
						copy(buffer, []byte(message))
						_, err = g.conn.localConn.WriteToUDP(buffer, g.conne.mcastAddress)
						check(err)

						g.recivedMsg[timeID] = message

					} else { //nick is ok, adding it to userNicks
						g.lastUserPing[msg[3]] = time.Now()
					}
					delete(g.lastUserPing, msg[2])
				} else {
					who = "You"
				}

				if msg[2] == tagNewName {
					fmt.Printf("\r*** %s has joined to chat ***\n", who)
					if i == 0 {
						g.lastUserPing[g.user] = time.Now()
						usage()
					}
				} else {
					if i == 0 {
						fmt.Printf("\r*** %s changed name to %s ***\n", who, msg[3])
					} else {
						fmt.Printf("\r*** %s changed name to %s ***\n", msg[2], msg[3])
					}
				}
				fmt.Print("\r<- ")
			}
		case commNickExist:
			{
				i := strings.Compare(addr.String(), g.conn.localConn.LocalAddr().String())
				if msg[1] == g.user && (i != 0) { //nick is the same, ip addr is not

					delete(g.lastUserPing, g.user)
					newName := "User" + strconv.Itoa(myRand.Intn(1000))
					fmt.Printf("\rSYSTEM: Nick %s already exists. Changing to %s\n", g.user, newName)
					fmt.Printf("SYSTEM: %s\n", usageChangeNick)
					fmt.Print("<- ")

					timeId := time.Now()

					message := fmt.Sprintf("%s:%s:%s:%s", timeID, commMyNick, name, newName)

					name = newName
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))
					_, err = g.conn.localConn.WriteToUDP(buffer, g.conn.mcastAddress)
					check(err)
				}
			}
		case commPrivate:
			{
				i := strings.Compare(addr.String(), g.conn.localConn.LocalAddr().String())
				if i != 0 {
					if len(msg) > 3 && msg[2] == g.user {
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
		case commPing:
			{
				userNames[msg[2]] = time.Now()
			}
		case commGotMsg:
			{
				delete(g.recivedMsg, msg[1])
			}
		default:
			{
				fmt.Println("you've got a strange message; ignore")
			}

		}
	}
}
