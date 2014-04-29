package main

import (
  "fmt"
  "crypto/md5")

type Node struct {
  one *Node
  zer *Node
  terminal int
}

func ExpandTree(root *Node, code []byte, value int) {
  var i, j, k uint
  var bit byte
  var newNode *Node
  curNode := root
  for i = 0; i<256; i++ {
    j = i/8
    k = 7-(i%8)

    bit = (code[j] >> k) & 1;

    if bit == 0 {
      newNode = curNode.zer
    } else {
      newNode = curNode.one
    }

    if newNode == nil {
      newNode = new(Node)
      newNode.terminal = 0
    }

    curNode = newNode
  }

  curNode.terminal = value
}

func smallHash(arr []byte) int {
  result := md5.Sum(arr);
  return int(result[len(result)-1])
}

func Nhash(s []byte) []byte {
  var counter [256]int
  for i, _ := range counter { counter[i] = 0 }

  for i, _ := range s {
    if i > len(s) - 4 { break }
    idx := smallHash(s[i:i+3])
    counter[idx]++
  }

  sum := 0
  for _, v := range counter { sum += v }
  mean := float32(sum) / 256.

  var code [32]byte
  for i, _ := range code {
    for k := 0; k<8; k++ {
      var orer byte
      if float32(counter[i*8+k]) >= mean {
        orer = 1
      } else {
        orer = 0
      }

      code[i] = (code[i] << 1) | orer
    }
  }

  return code[:]
}

func main() {
  SearchTree := new(Node)

  str := []byte("Sup brajadoodle do.")
  // str2 := []byte("Sup brojadoodle do?")
  code1 := Nhash(str)
  ExpandTree(SearchTree, code1, 42)
  fmt.Println(code1)
}
