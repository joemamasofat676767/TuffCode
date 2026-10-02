package main

import (
	"os"
	"io"
	"fmt"
	"slices"
	"errors"
	"os/exec"
	"strings"
	"path/filepath"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/blacknon/tvxterm"
)

func main(){
	logFile, err := os.OpenFile(".LOGS", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	keyLogsFile, err := os.OpenFile(".KEYLOGS", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	var dir_temp string
	if err != nil{
		panic(err)
	}
	fmt.Fprintln(logFile,"app run")
	var dir = []string{"."}
	var content string
	for{
		files, err := os.ReadDir(strings.Join(dir, "/"))
		fmt.Println(strings.Join(dir, "/"))
		if err != nil{
			fmt.Fprintf(logFile, "error: %v\n", err)
			dir = []string{"."}
		}
		fmt.Println(files)
		fmt.Println("choose path")
		_, err = fmt.Scan(&dir_temp)
		info, err := os.Stat(strings.Join(append(dir, dir_temp), "/"))
		if err != nil{
			fmt.Fprintf(logFile, "error: %v\n", err)
		}
		if _, err := os.Stat(filepath.Join(strings.Join(dir, "/"), dir_temp)); errors.Is(err, os.ErrNotExist){
			fmt.Println("no file or dir: ", dir_temp)
			continue
		}
		if !info.IsDir(){
			content_byte, err := os.ReadFile(strings.Join(append(dir, dir_temp), "/"))
			content = string(content_byte)
			if err != nil{
				fmt.Fprintf(logFile, "error: %v\n", err)
			}
			fmt.Println("use this file? [Y/N]: ")
			var choice string
			_, err = fmt.Scan(&choice)
			if err != nil{
				fmt.Fprintf(logFile, "error: %v\n", err)
			}
			if choice == "Y" || choice == "y"{
				break
			}
		}else{
			if dir_temp == ".."{
				if len(dir) != 1{
					slices.Delete(dir, len(dir)-1, len(dir))
				}else{
					fmt.Println("cant do that, limited to this directory only")
				}
			}else{
				dir = append(dir, dir_temp)
			}
		}
	}

	mode := "e"

	app := tview.NewApplication().EnableMouse(true)

	textEditor := tview.NewTextArea().SetPlaceholder("enter your code")
	textEditor.SetBorder(true).SetTitle(" TuffCode ")
	textEditor.SetText(string(content), false)

	terminal := tvxterm.New(app)
	terminal.SetBorder(true).SetTitle(" terminal ")
	cmd := exec.Command("/data/data/com.termux/files/usr/bin/bash", "-i")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	backend, err := tvxterm.NewPTYBackend(cmd, 80, 24)
	if err != nil{
		fmt.Fprintf(logFile, "error: %v\n", err)
		panic(err)
	}
	fmt.Fprintf(logFile, "backend: %v\n", backend)
	terminal.Attach(backend)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey{
		if event.Key() == tcell.KeyRune{
			fmt.Fprintf(keyLogsFile, "key: %v\n", string(event.Rune()))
		}else{
			fmt.Fprintf(keyLogsFile, "key: %v\n", tcell.KeyNames[event.Key()])
		}
		if event.Key() == tcell.KeyCtrlT{
			if mode == "e"{
				mode = "t"
				app.SetRoot(terminal, true)
				app.SetFocus(terminal)
			}else{
				mode = "e"
				app.SetRoot(textEditor, true)
				app.SetFocus(textEditor)
			}
			return nil
		}
		if event.Key() == tcell.KeyCtrlQ{
			app.Stop()
			return nil
		}
		if event.Key() == tcell.KeyCtrlS{
			err := os.WriteFile(dir_temp, []byte(textEditor.GetText()), 0644)
			if err != nil{
				fmt.Fprintf(logFile, "saving error: %v\n", err)
			}
		}

		if mode == "e"{
			if event.Key() == tcell.KeyCtrlC{
				if textEditor.HasSelection(){
					selectedText, _, _ := textEditor.GetSelection()
					copyCommand1, copyCommand2 := exec.Command("cat"), exec.Command("termux-clipboard-set")
					copyCommand1.Stdin = strings.NewReader(selectedText)
					r, w := io.Pipe()
					copyCommand1.Stdout = w
					copyCommand2.Stdin = r
					if copyCommand1.Start() ; err != nil{
						fmt.Fprintf(logFile, "error: %v\n", err)
					}
					if copyCommand2.Start() ; err != nil{
						fmt.Fprintf(logFile, "error: %v\n", err)
					}
					go func(){
						defer r.Close()
						copyCommand1.Wait()
					}()
				}
				return nil
			}else if event.Key() == tcell.KeyCtrlV{
				pasteCmd := exec.Command("sh", "-c", "termux-clipboard-get")
				pasteTextB, err := pasteCmd.Output()
				if err != nil{
					fmt.Fprintf(logFile, "error: %v\n", err)
				}
				pasteText := string(pasteTextB)
				if paste := textEditor.PasteHandler() ; paste != nil{
					paste(pasteText, nil)
				}
				return nil
			}else if event.Modifiers() == tcell.ModAlt{
				row, col, _, _ := textEditor.GetCursor()
				editorText := strings.Split(textEditor.GetText(), "\n")
				if event.Key() == tcell.KeyUp{
					if row <= 0{
						return nil
					}
					textEditor.SetText(strings.Join(editorText[:row-1], "\n")+"\n"+editorText[row]+"\n"+editorText[row-1]+"\n"+strings.Join(editorText[row+1:], "\n")+"\n", false)
					textEditor.Select(col, col)
					for i := 0; i < row-1; i++{
						textEditor.InputHandler()(
							tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
							func(p tview.Primitive){},
						)
					}
					return nil
				}
				if event.Key() == tcell.KeyDown{
					if row >= len(editorText)-2{
						return nil
					}
					textEditor.SetText(strings.Join(editorText[:row], "\n")+"\n"+editorText[row+1]+"\n"+editorText[row]+"\n"+strings.Join(editorText[row+2:], "\n")+"\n", false)
					textEditor.Select(col, col)
					for i := 0; i < row + 1; i++{
						textEditor.InputHandler()(
							tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
							func(p tview.Primitive){},
						)
					}
					return nil
				}
			}
		}
		return event
	})

	if err := app.SetRoot(textEditor, true).Run(); err != nil {
		panic(err)
	}
}
