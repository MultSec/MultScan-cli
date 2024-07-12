package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
    "io"
    "mime/multipart"
	"strings"

	"github.com/mgutz/ansi"
)

// Define a struct for the known parts of the JSON structure
type Machine struct {
    MachineIP   string `json:"machine_ip"`
    MachineName string `json:"machine_name"`
}

// Struct to unmarshal JSON fileinfo
type Info struct {
    Digests        []string          `json:"digests"`
    PublicPresence map[string]bool   `json:"public_presence"`
    Size           string            `json:"size"`
    Type           string            `json:"type"`
}

type FileTypeResponse struct {
    Info Info `json:"info"`
}

type Log int64

const (
    logError Log = iota
    logInfo
    logStatus
    logInput
	logSuccess
	logSection
	logSubSection
)

// Function to print logs
func printLog(log Log, text string) {
	switch log {
	case logError:
		fmt.Printf("[%s] %s %s\n", ansi.ColorFunc("red")("!"), ansi.ColorFunc("red")("ERROR:"), ansi.ColorFunc("cyan")(text))
	case logInfo:
		fmt.Printf("[%s] %s\n", ansi.ColorFunc("blue")("i"), text)
	case logStatus:
		fmt.Printf("[*] %s\n", text)
	case logInput:
		fmt.Printf("[%s] %s", ansi.ColorFunc("yellow")("?"), text)
	case logSuccess:
		fmt.Printf("[%s] %s\n", ansi.ColorFunc("green")("+"), text)
	case logSection:
		fmt.Printf("\t[%s] %s\n", ansi.ColorFunc("yellow")("-"), text)
	case logSubSection:
		fmt.Printf("\t\t[%s] %s\n", ansi.ColorFunc("magenta")(">"), text)
	}
}

// Function to get machines from the server
func getMachines(ip string, port int) ([]Machine, error) {
	var machines []Machine

	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Server IP: "), ansi.ColorFunc("cyan")(ip)))
	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Server Port: "), ansi.ColorFunc("cyan")(fmt.Sprintf("%d", port))))

	url := fmt.Sprintf("http://%s:%d/api/v1/machines", ip, port)
	resp, err := http.Get(url)
	if err != nil {
		return machines, fmt.Errorf("failed to fetch machines: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return machines, fmt.Errorf("server returned non-200 status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return machines, fmt.Errorf("failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, &machines); err != nil {
		return machines, fmt.Errorf("failed to parse known parts of JSON: %v", err)
	}

	return machines, nil
}

// Function to display machines in a readable format
func displayMachines(machines []Machine) {
	printLog(logInfo, "Retrieved machines from server")
	for _, machine := range machines {
		printLog(logSection, fmt.Sprintf("%s:", ansi.ColorFunc("default+hb")(machine.MachineName)))
		printLog(logSubSection, fmt.Sprintf("%s", ansi.ColorFunc("cyan")(machine.MachineIP)))
    }
}

// Generate UUID to be used for sample processing
func generateID() (string, error) {
	// Generate a random 16-bit integer
	var randomInt uint16
	err := binary.Read(rand.Reader, binary.LittleEndian, &randomInt)
	if err != nil {
		return "", fmt.Errorf("error generating random 16-bit integer: %w", err)
	}

	// Convert the random integer to a byte slice
	randomBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(randomBytes, randomInt)

	// Hash the byte slice using SHA-256
	hasher := sha256.New()
	_, err = hasher.Write(randomBytes)
	if err != nil {
		return "", fmt.Errorf("error hashing the random bytes: %w", err)
	}
	hash := hasher.Sum(nil)

	// Encode the hash in Base32
	base32Hash := base32.StdEncoding.EncodeToString(hash)

	return base32Hash, nil
}

func UploadMultipartFile(client *http.Client, uri, key, path string) (*http.Response, error) {
    body, writer := io.Pipe()

    req, err := http.NewRequest(http.MethodPost, uri, body)
    if err != nil {
        return nil, err
    }

    mwriter := multipart.NewWriter(writer)
    req.Header.Add("Content-Type", mwriter.FormDataContentType())

    errchan := make(chan error)

    go func() {
        defer close(errchan)
        defer writer.Close()
        defer mwriter.Close()

        w, err := mwriter.CreateFormFile(key, path)
        if err != nil {
            errchan <- err
            return
        }

        in, err := os.Open(path)
        if err != nil {
            errchan <- err
            return
        }
        defer in.Close()

        if written, err := io.Copy(w, in); err != nil {
            errchan <- fmt.Errorf("error copying %s (%d bytes written): %v", path, written, err)
            return
        }

        if err := mwriter.Close(); err != nil {
            errchan <- err
            return
        }
    }()

    resp, err := client.Do(req)
    merr := <-errchan

    if err != nil || merr != nil {
        return resp, fmt.Errorf("http error: %v, multipart error: %v", err, merr)
    }

    return resp, nil
}

func sendSample(ip string, port int, id string, sampleFile string) error {
	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Sample file: "), ansi.ColorFunc("cyan")(sampleFile)))
	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Client ID: "), ansi.ColorFunc("cyan")(id)))

	// Define the URI
	uri := fmt.Sprintf("http://%s:%d/api/v1/sample/upload/%s", ip, port, id)

	// Server expect key to be sample
	key := "sample"

	client := &http.Client{}

	// Upload the file to the server
	resp, err := UploadMultipartFile(client, uri, key, sampleFile)
	if err != nil {
		printLog(logError, fmt.Sprintf("Failed to upload sample: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		printLog(logError, fmt.Sprintf("Failed to upload sample, server responded with status: %s", resp.Status))
		return err
	}

	printLog(logInfo, fmt.Sprintf("%s", ansi.ColorFunc("default+hb")("Sample uploaded successfully")))

	return nil
}

// Requests fileinfo from server
func requestFileInfo(ip string, port int, id string) error {
	fileInfoUri := fmt.Sprintf("http://%s:%d/api/v1/sample/fileinfo/%s", ip, port, id)

	// Make GET request on fileInfoUri
	resp, err := http.Get(fileInfoUri)
	if err != nil {
		printLog(logError, fmt.Sprintf("%v", err))
		return err
	}

	defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        err = fmt.Errorf("received non-200 response status: %d", resp.StatusCode)
        printLog(logError, fmt.Sprintf("%v", err))
        return err
    }

    // Read the response body
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        printLog(logError, fmt.Sprintf("Error reading response body: %v", err))
        return err
    }

    // Unmarshal the JSON response
    var filetypeResponse FileTypeResponse
    err = json.Unmarshal(body, &filetypeResponse)
    if err != nil {
        printLog(logError, fmt.Sprintf("Error unmarshaling JSON: %v", err))
        return err
    }

    // Log the filetype information
	printLog(logSuccess, fmt.Sprintf("%s", ansi.ColorFunc("default+hb")("File information")))

    // Show filetype
    printLog(logSection, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Filetype:"), ansi.ColorFunc("cyan")(filetypeResponse.Info.Type)))

    // Show digests
    printLog(logSection, ansi.ColorFunc("default+hb")("Digests"))
    for _, digest := range filetypeResponse.Info.Digests {
		parts := strings.Split(digest, ":")

		// Access the parts
		algorithm := parts[0]
		hash := parts[1]

        printLog(logSubSection, fmt.Sprintf("%s: %s", ansi.ColorFunc("default+hb")(algorithm), ansi.ColorFunc("cyan")(hash)))
    }

    // Show file public presence
    printLog(logSection, ansi.ColorFunc("default+hb")("Public Presence"))
    for key, value := range filetypeResponse.Info.PublicPresence {
		if value {
			printLog(logSubSection, fmt.Sprintf("%s: %s", ansi.ColorFunc("default+hb")(key) , "🟢"))
		} else {
			printLog(logSubSection, fmt.Sprintf("%s: %s", ansi.ColorFunc("default+hb")(key) , "🔴"))
		}
    }
	
	return nil
}

/*
func requestScanStatus(ip string, port int, id string) error {
	sampleScanUri := fmt.Sprintf("http://%s:%d/api/v1/sample/scan/%s", ip, port, id)

}
*/

func requestSampleDeletion(ip string, port int, id string) error {
	sampleDeletionUri := fmt.Sprintf("http://%s:%d/api/v1/sample/delete/%s", ip, port, id)

	// Make GET request on sampleDeletionUri
	resp, err := http.Get(sampleDeletionUri)
	if err != nil {
		printLog(logError, fmt.Sprintf("%v", err))
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		printLog(logError, fmt.Sprintf("Failed to delete sample, server responded with status: %s", resp.Status))
		return err
	}

	printLog(logInfo, fmt.Sprintf("%s", ansi.ColorFunc("default+hb")("Sample deleted successfully")))

	return nil
}