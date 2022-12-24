/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"
  "io"
  "fmt"
  "errors"
  "net/http"
  "regexp"
  "time"

  "github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
  "github.com/tidwall/gjson"
  "github.com/cavaliergopher/grab/v3"
  "github.com/artdarek/go-unzip"
  "github.com/gosuri/uilive"
)

type Album struct {
  Artist string
  Title string
  Url string
}

var (
  format  string
)

var rootCmd = &cobra.Command{
  Use:   "bandcamp-dl",
  Short: "Downloads items from bandcamp purchase pages",
  Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
  Run: func(cmd *cobra.Command, args []string) { 
    getDownloads(args[0])
  },
}

var titleRe = regexp.MustCompile(`[\/,:]`)

var client = grab.NewClient()

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&format, "format", "alac", "format to download items. default is ALAC")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getDownloads(url string) {
  resp, err := http.Get(url)

  if err != nil {
    fmt.Println("failed to fetch data")
    return
  }

  defer resp.Body.Close()
  if resp.StatusCode != 200 {
    fmt.Printf("status code error: %d %s", resp.StatusCode, resp.Status)
  }

  parseHtml(resp.Body)
}

func parseHtml(body io.ReadCloser) {
  doc, err := goquery.NewDocumentFromReader(body)
  if err != nil {
    fmt.Println(err)
  }
  val, exists := doc.Find("#pagedata").Attr("data-blob")
  if !exists {
    fmt.Printf("data does not exist")
  }
  // fmt.Println(val)
  parseJSON(val)
}

func parseJSON(val string) {
  var data []Album

  downloadStr := fmt.Sprintf("downloads.%s.url", format)

  result := gjson.Get(val, "download_items")
  result.ForEach(func(key, value gjson.Result) bool {
    var album Album
    album.Artist = value.Get("artist").String()
    album.Title = value.Get("title").String()
    album.Url = value.Get(downloadStr).String()
    data = append(data, album)
    return true
  })

  downloadItems(data)
}

func downloadItems(payload []Album) {
  for _, value := range payload {
    download(value)
  }
}

// func downloadMultiple(items []DownloadItem) {
//   for i:= 0; i < len(items); i++{
//     if (len(items[i].tralbums) > 0) {
//       for t := 0; t < len(items[i].tralbums); t++ {
//         downloadItem(items[i].tralbums[t])
//       }
//     }
//   }
// }

func download(item Album) {
  filestring := fmt.Sprintf("%s - %s", item.Artist, item.Title)
  filestring = titleRe.ReplaceAllString(filestring, "-")
  zip := filestring + ".zip"

  if _, err := os.Stat(zip); errors.Is(err, os.ErrNotExist) {
    doDownload(item.Url)
  } else {
    if _, err := os.Stat(filestring); errors.Is(err,os.ErrNotExist) {
      doUnzip(zip, filestring)
    }
  }
}

func doDownload(url string) {
  req, _ := grab.NewRequest(".", url)

  writer := uilive.New()
  writer.Start()
  fmt.Fprintf(writer, "Downloading %v...\n", req.URL())
  resp := client.Do(req)
  
  var i int64
  i = 0
  for i < resp.Size() {
    fmt.Fprintf(writer, "transferred %v / %v bytes (%.2f%%)\n",
      resp.BytesComplete(),
      resp.Size(),
      resp.Progress()*100)
    time.Sleep(time.Millisecond * 500)
    i = resp.BytesComplete()
  }

  // check for errors
  if err := resp.Err(); err != nil {
    fmt.Fprintf(writer, "Download failed: %v\n", err)
    os.Exit(1)
  }

  fmt.Fprintf(writer, "Download saved to ./%v \n", resp.Filename)
  writer.Stop()
}

func doUnzip(src string, dest string) {
  uz := unzip.New(src, dest)

  uz.Extract()
}

func rmDownload(file string) {
  err := os.Remove(file)
  if err != nil {
    fmt.Println(err)
  }
}
