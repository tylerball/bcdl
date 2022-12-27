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
  "strings"
  // "sync"

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
  format        string
  keepArchives  bool
  downloadStr   string
)

var Formats = []string{"mp3-v0", "mp3-320", "flac", "aac-hi", "vorbis", "alac", "wav", "aiff-lossless"}
var titleRe = regexp.MustCompile(`[\/,:]`)

var rootCmd = &cobra.Command{
  Use:   "bandcamp-dl [flags] [url]",
  Short: "Downloads items from bandcamp purchase pages",
  Long: `bcdl handles downloading purchaes from bandcamp
download pages.`,
  Args: func(cmd *cobra.Command, args []string) error {
    if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
      return err
    }
    if !checkFormat() {
      return errors.New(fmt.Sprintf("Invalid format. Must be one of: %s", strings.Trim(fmt.Sprint(Formats), "[]")))
    }
    return nil
  },
  Run: func(cmd *cobra.Command, args []string) {
    getDownloads(args[0])
  },
}

var client = grab.NewClient()

func init() {
  rootCmd.PersistentFlags().StringVar(&format, "format", "alac", "format to download items. default is ALAC")
  rootCmd.PersistentFlags().BoolVarP(&keepArchives, "keep-archives", "D", false, "Keep zip files after extraction?")
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func checkFormat() bool {
  var result bool = false
  for i := 0; i < len(Formats); i++ {
    if Formats[i] == format {
      result = true
      break
    }
  }
  downloadStr = fmt.Sprintf("downloads.%s.url", format)
  return result
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

  result := gjson.Get(val, "download_items")
  result.ForEach(func(key, value gjson.Result) bool {
    var tralbums = value.Get("tralbums.#").Int()
    var t int64
    if tralbums > 0 {
      for t = 0; t < tralbums; t++ {
        data = append(data, parseAlbum(value.Get(fmt.Sprintf("tralbums.%d", t))))
      }
    } else {
      data = append(data, parseAlbum(value))
    }
    return true
  })

  downloadItems(data)
}

func parseAlbum(value gjson.Result) Album {
  var album Album
  album.Artist = value.Get("artist").String()
  album.Title = value.Get("title").String()
  album.Url = value.Get(downloadStr).String()
  fmt.Println(album.Url)
  return album
}

func downloadItems(payload []Album) {
  // var wG sync.WaitGroup

  for _, value := range payload {
    // wG.Add(1)
    // download(value, &wG)
    download(value)
  }

  // wG.Wait()
}

func download(item Album) {
  filestring := fmt.Sprintf("%s - %s", item.Artist, item.Title)
  filestring = titleRe.ReplaceAllString(filestring, "-")
  zip := filestring + ".zip"

  if _, err := os.Stat(zip); errors.Is(err, os.ErrNotExist) {
    doDownload(item.Url)
    doUnzip(zip, filestring)
    if !keepArchives {
      rmDownload(zip)
    }
  } else {
    if _, err := os.Stat(filestring); errors.Is(err,os.ErrNotExist) {
      doUnzip(zip, filestring)
      if !keepArchives {
        rmDownload(zip)
      }
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
