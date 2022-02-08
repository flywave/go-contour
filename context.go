package contour

import "sync"

type Context struct {
	writer PolygonWriter
	idx    int64
	idlock sync.Mutex
}
