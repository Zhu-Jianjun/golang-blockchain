package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
)

var localMessagePool = []Message{}

type node struct {
	nodeID string
	addr string //node address for listening
	rsaPrivKey []byte
	rsaPubKey []byte
}

type pbft struct {
	//info of node
	node node
	//self-increasing after each request
	sequenceID int
	lock sync.Mutex
	//temporary message pool
	messagePool map[string]Request
	//to store Prepare messages(at least 2f)
	prePareConfirmCount map[string]map[string]bool
	//to store Commit messages(at least 2f+1)
	commitConfirmCount map[string]map[string]bool
	//whether Commit is done
	isCommitBordcast map[string]bool
	//whether Reply is done
	isReply map[string]bool
}

func NewPBFT(nodeID, addr string) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = p.getPivKey(nodeID) 
	p.node.rsaPubKey = p.getPubKey(nodeID)  
	p.sequenceID = 0
	p.messagePool = make(map[string]Request)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	return p
}

func (p *pbft) handleRequest(data []byte) {
	//Cut the message and invoke different functions according to the message command
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case cRequest:
		p.handleClientRequest(content)
	case cPrePrepare:
		p.handlePrePrepare(content)
	case cPrepare:
		p.handlePrepare(content)
	case cCommit:
		p.handleCommit(content)
	}
}

//to process the request from the client
func (p *pbft) handleClientRequest(content []byte) {
	fmt.Println("The primary node has received the request from the client.")
	//The Request structure is parsed using JSON
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//to add infoID
	p.sequenceIDAdd()
	//to get the digest
	digest := getDigest(*r)
	fmt.Println("The request has been stored into the temporary message pool.")
	//to store into the temp message pool
	p.messagePool[digest] = *r
	//to sign the digest by the primary node
	digestByte, _ := hex.DecodeString(digest)
	signInfo := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
	//setup PrePrepare message and send to other nodes
	pp := PrePrepare{*r, digest, p.sequenceID, signInfo}
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("sending PrePrepare messsage to all the other nodes...")
	//to send PrePrepare message to other nodes
	p.broadcast(cPrePrepare, b)
	fmt.Println("PrePrepare is done.")
}

//to process the PrePrepare message
func (p *pbft) handlePrePrepare(content []byte) {
	fmt.Println("This node has received the PrePrepare message from the primary node.")
	//The Request structure is parsed using JSON
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}
	//to get the public key of the master node for digital signature verification
	primaryNodePubKey := p.getPubKey("N0")
	digestByte, _ := hex.DecodeString(pp.Digest)
	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("The digest is not correct. Deny sending Prepare messages.")
	} else if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("ID is not correct. Deny sending Prepare messages.")
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		fmt.Println("The signiture of primary node is not valid! Deny sending Prepare messages.")
	} else {
		
		p.sequenceID = pp.SequenceID
		
		fmt.Println("The PrePrepare message has been stored into the temporary message pool.")
		p.messagePool[pp.Digest] = pp.RequestMessage
		
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID, sign}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		
		fmt.Println("sending Prepare messages to other nodes...")
		p.broadcast(cPrepare, bPre)
		fmt.Println("Prepare is done.")
	}
}

//to process Prepare message
func (p *pbft) handlePrepare(content []byte) {
	//The Request structure is parsed using JSON
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("This node has received the Prepare message from %s ... \n", pre.NodeID)
	//
	MessageNodePubKey := p.getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("The current temporary message pool does not have this digest. Deny sending Commit message.")
	} else if p.sequenceID != pre.SequenceID {
		fmt.Println("ID is not correct. Deny sending Commit message.")
	} else if !p.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		fmt.Println("The signiture is not valid! Deny sending Commit message.")
	} else {
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//Since the primary node does not send Prepare message, so it does not include itself.
		specifiedCount := 0
		if p.node.nodeID == "N0" {
			specifiedCount = nodeCount / 3 * 2
		} else {
			specifiedCount = (nodeCount / 3 * 2) - 1
		}
		
		p.lock.Lock()
		
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			fmt.Println("This node has received at least 2f (including itself) Prepare messages.")
			
			sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			c := Commit{pre.Digest, pre.SequenceID, p.node.nodeID, sign}
			bc, err := json.Marshal(c)
			if err != nil {
				log.Panic(err)
			}
			
			fmt.Println("sending Commit message to other nodes...")
			p.broadcast(cCommit, bc)
			p.isCommitBordcast[pre.Digest] = true
			fmt.Println("Commit is done.")
		}
		p.lock.Unlock()
	}
}

//to process Commit message
func (p *pbft) handleCommit(content []byte) {
	//The Request structure is parsed using JSON
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("This node has received Commit message from %s. \n", c.NodeID)
	
	MessageNodePubKey := p.getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		fmt.Println("The current temporary message pool does not have this digest. Deny storing into local message pool.")
	} else if p.sequenceID != c.SequenceID {
		fmt.Println("ID is not correct. Deny storing into local message pool.")
	} else if !p.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		fmt.Println("The signiture is not valid! Deny storing into local message pool.")
	} else {
		p.setCommitConfirmMap(c.Digest, c.NodeID, true) 
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		
		p.lock.Lock()
		if count >= nodeCount/3*2 && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			fmt.Println("This node has received at least 2f+1 (including itself) Commit messages.")
			
			localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
			info := p.node.nodeID + " has stored the message with msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + " into the local message pool successfully. The message is " + p.messagePool[c.Digest].Content
			
			fmt.Println(info)
			fmt.Println("sending Reply message to the client ...")
			tcpDial([]byte(info), p.messagePool[c.Digest].ClientAddr)
			p.isReply[c.Digest] = true
			fmt.Println("Reply is done.")
		}
		p.lock.Unlock()
	}
}


func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

//to send to other nodes
func (p *pbft) broadcast(cmd command, content []byte) {
	for i := range nodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := jointMessage(cmd, content)
		go tcpDial(message, nodeTable[i])
	}
}


func (p *pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}


func (p *pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}


func (p *pbft) getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}


func (p *pbft) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}
