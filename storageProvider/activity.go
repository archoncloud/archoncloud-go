package storageProvider

import (
	. "github.com/itsmeknt/archoncloud-go/common"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	fp "path/filepath"
	"sync"
)

// Note: this is a temporary solution as the activity files can be tampered with

type ActivityInfo struct {
	Upload bool
	Url *ArchonUrl
	Client string
	NumBytes int64
}

/*
type ActivityReport struct {
	Year int
	Month int
	Upload string	`json:",omitempty"`
	Download string	`json:",omitempty"`
}

const (
	uploadPrefix = "u:"
	downloadPrefix = "d:"
)
*/

var activityMutex sync.Mutex
var activityLogger *log.Logger

func RecordActivity(a *ActivityInfo) {
	{
		activityMutex.Lock()
		defer activityMutex.Unlock()
		if activityLogger == nil {
			l := lumberjack.Logger{
				// Filename is the file to write logs to.  Backup log files will be retained
				// in the same directory.  It uses <processname>-lumberjack.log in
				// os.TempDir() if empty.
				Filename:   fp.Join(GetActivityFolder(), "activity.log"),
				// MaxSize is the maximum size in megabytes of the log file before it gets
				// rotated. It defaults to 100 megabytes.
				MaxSize:    50, // megabytes
				// MaxBackups is the maximum number of old log files to retain.  The default
				// is to retain all old log files (though MaxAge may still cause them to get
				// deleted.)
				//MaxBackups: 6,
				// MaxAge is the maximum number of days to retain old log files based on the
				// timestamp encoded in their filename.  Note that a day is defined as 24
				// hours and may not exactly correspond to calendar days due to daylight
				// savings, leap seconds, etc. The default is not to remove old log files
				// based on age.
				MaxAge:     31*3, //days
				// Compress determines if the rotated log files should be compressed
				// using gzip.
				Compress:   false,
				// LocalTime determines if the time used for formatting the timestamps in
				// backup files is the computer's local time.  The default is to use UTC
				// time.
			}
			activityLogger = log.New(&l, "", log.LstdFlags)
		}
	}

	//	02-04-2020 3:30pm - storage request for "arc://archon.af/demo.jpg" from IP address 212.12.500.9000:8000 - shards 2 and 6 downloaded (2560 bytes downloaded)
	direction := "d"
	if a.Upload {
		direction = "u"
	}
	activityLogger.Printf("%s %d %s %s\n", direction, a.NumBytes, a.Url.String(), a.Client);
}

/*
func GetActivityReport(perm pl.PermissionLayerID, monthsBack int) (rep []ActivityReport) {
	if monthsBack >= 0 {
		now := time.Now().UTC()
		year, dmonth, _ := now.Date()
		month := int(dmonth)
		for {
			a := activityReportFor(perm, year, month)
			if a.Year != 0 {
				rep = append(rep, a)
			}
			monthsBack--
			if monthsBack < 0 {
				break
			}
			month--
			if month < 0 {
				month = 12
				year--
			}
		}
	}
	return
}

func activityReportFor(perm pl.PermissionLayerID, year, month int) ActivityReport {
	var uploadBytes, downloadBytes int64
	rep := ActivityReport{}
	path := fp.Join(GetActivityFolder(string(perm)), activityFileName(year, month))
	activityMutex.Lock()
	defer activityMutex.Unlock()
	file, err := os.Open(path)
	if err == nil {
		rep.Year = year
		rep.Month = month
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if err = scanner.Err(); err != nil {break}
			line := scanner.Text()
			var p *int64
			if strings.HasPrefix(line, uploadPrefix) {
				p = &uploadBytes
			} else if strings.HasPrefix(line, downloadPrefix) {
				p = &downloadBytes
			} else {
				break
			}
			v, err := strconv.ParseInt(line[2:], 10, 64)
			if err != nil {break}
			*p += v
		}
		if uploadBytes != 0 {
			rep.Upload = humanize.Bytes(uint64(uploadBytes))
		}
		if downloadBytes != 0 {
			rep.Download = humanize.Bytes(uint64(downloadBytes))
		}
	} else {
		LogError.Println(err)
	}
	return rep
}

func activityFileName(year, month int) string {
	return fmt.Sprintf("%04d-%02d.txt", year, month)
}

func currentActivityFileName() string {
	now := time.Now().UTC()
	year, month, _ := now.Date()
	return activityFileName(year, int(month))
}
*/
