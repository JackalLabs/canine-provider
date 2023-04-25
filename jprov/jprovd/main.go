package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

type MyData struct {
	Cid string `json:"cid"`
	Fid string `json:"fid"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "My App does amazing things!",
		Long:  "A longer description of what My App does and how it works.",

		// Default command definition
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}

	// Register the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	target_url := "https://biphan.cloud/upload"
	_, resp := postFile("test3.txt", target_url)
	fmt.Println(resp)

	var data MyData
	err := json.Unmarshal([]byte(resp), &data)
	if err != nil {
		panic(err)
	}

	fmt.Println(data.Cid)

	clientCtx := client.GetClientContextFromCmd(rootCmd)
	updatedclientCtx := clientCtx.WithChainID("jackal-1").WithBroadcastMode(flags.BroadcastBlock).WithOutputFormat("json").WithFrom("jkl1kuzaqt6ue7wqz595kxp5ejxhkzwg0yxkdfz9ys")
	fmt.Println(updatedclientCtx)

	msg := storagetypes.NewMsgSignContract(
		"jkl1kuzaqt6ue7wqz595kxp5ejxhkzwg0yxkdfz9ys",
		data.Cid,
		true,
	)
	res, err := utils.SendTx(updatedclientCtx, rootCmd.Flags(), fmt.Sprintf("Storage Provided by %s", "A Storage Provider"), msg)
	fmt.Println(res.Height)
	fmt.Println(res.Info)
	fmt.Println(res.RawLog)

	// tx.GenerateOrBroadcastTxCLI(updatedclientCtx, rootCmd.Flags(), msg)
}

func postFile(filename string, targetUrl string) (error, string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return err, ""
	}
	// I need to pass in a sender mang lol
	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return err, ""
	}
	defer fh.Close()

	// iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		fmt.Println("error copying")
		return err, ""
	}
	// Below might need to change - we can look at how HTML does it to replicate it
	bodyWriter.WriteField("sender", "jkl1kuzaqt6ue7wqz595kxp5ejxhkzwg0yxkdfz9ys")
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		fmt.Println("Post request from main.go failed.")
		fmt.Println(err)
		return err, ""
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, ""
	}
	fmt.Println(resp.Status)
	return nil, string(resp_body)
}
