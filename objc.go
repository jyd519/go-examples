package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
void
logtest(void) {
		int width = [[NSScreen mainScreen] frame].size.width;
		int height = [[NSScreen mainScreen] frame].size.height;
    NSLog(@"from objective-c %d, %d", width, height);
}

*/
import "C"

func main() {
	C.logtest()
}
