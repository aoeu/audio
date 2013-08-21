#include "portaudio_interop.h"
#include "_cgo_export.h"

PaStreamCallback* getStreamCallback() {
	return (PaStreamCallback*)streamCallback;
}