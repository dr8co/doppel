//go:build windows && cgo

package trash

/*
#cgo LDFLAGS: -lole32 -lshell32

#ifndef _WIN32_WINNT
#define _WIN32_WINNT 0x0A00
#endif
#ifndef WINVER
#define WINVER 0x0A00
#endif
#define COBJMACROS
#include <windows.h>
#include <objbase.h>
#include <shellapi.h>
#include <shobjidl.h>
#include <stdlib.h>

#ifndef FOF_WANTNUKEWARNING
#define FOF_WANTNUKEWARNING 0x4000
#endif
#ifndef FOFX_RECYCLEONDELETE
#define FOFX_RECYCLEONDELETE 0x00080000
#endif
// TSF_DELETE_RECYCLE_IF_POSSIBLE, as passed to PreDeleteItem.
#define TRASH_RECYCLE_IF_POSSIBLE 0x00000080

// GUIDs are spelled out so that no import library for IIDs is needed.
static const GUID k_IID_IUnknown   = {0x00000000, 0x0000, 0x0000, {0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}};
static const GUID k_IID_Sink       = {0x04b0f1a7, 0x9490, 0x44bc, {0x96, 0xe1, 0x42, 0x96, 0xa3, 0x12, 0x52, 0xe2}};
static const GUID k_CLSID_FileOp   = {0x3ad05575, 0x8857, 0x4850, {0x92, 0x77, 0x11, 0xb8, 0x5b, 0xdb, 0x8e, 0x09}};
static const GUID k_IID_FileOp     = {0x947aab5f, 0x0a5c, 0x4c13, {0xb4, 0xd6, 0x4b, 0xf7, 0x83, 0x6f, 0xc9, 0xf8}};
static const GUID k_IID_ShellItem  = {0x43826d1e, 0xe718, 0x42ee, {0xbc, 0x55, 0xa1, 0xe2, 0x61, 0xc3, 0x7b, 0xfe}};

// ---------------------------------------------------------------------------
// Progress sink. Its only job is a safety net: if the shell announces (in
// PreDeleteItem) that an item would be destroyed instead of recycled, we
// abort that operation, so a "move to trash" can never become a permanent
// delete. It also records the per-item result from PostDeleteItem, because
// PerformOperations() may return S_OK although the item failed.
// ---------------------------------------------------------------------------
typedef struct {
	IFileOperationProgressSink iface; // must be first
	LONG    refs;
	int     refused;
	int     saw_post;
	HRESULT delete_hr;
} Sink;

static ULONG STDMETHODCALLTYPE sink_AddRef(IFileOperationProgressSink *t) {
	return (ULONG)InterlockedIncrement(&((Sink *)t)->refs);
}

static ULONG STDMETHODCALLTYPE sink_Release(IFileOperationProgressSink *t) {
	LONG r = InterlockedDecrement(&((Sink *)t)->refs);
	if (r == 0) {
		free(t);
	}
	return (ULONG)r;
}

static HRESULT STDMETHODCALLTYPE sink_QueryInterface(IFileOperationProgressSink *t, REFIID riid, void **ppv) {
	if (ppv == NULL) {
		return E_POINTER;
	}
	if (IsEqualGUID(riid, &k_IID_IUnknown) || IsEqualGUID(riid, &k_IID_Sink)) {
		*ppv = t;
		sink_AddRef(t);
		return S_OK;
	}
	*ppv = NULL;
	return E_NOINTERFACE;
}

static HRESULT STDMETHODCALLTYPE sink_PreDeleteItem(IFileOperationProgressSink *t, DWORD flags, IShellItem *item) {
	if (!(flags & TRASH_RECYCLE_IF_POSSIBLE)) {
		((Sink *)t)->refused = 1;
		return E_ABORT;
	}
	return S_OK;
}

static HRESULT STDMETHODCALLTYPE sink_PostDeleteItem(IFileOperationProgressSink *t, DWORD flags, IShellItem *item, HRESULT hr, IShellItem *created) {
	((Sink *)t)->saw_post = 1;
	((Sink *)t)->delete_hr = hr;
	return S_OK;
}

#define SINK_STUB(name, params) \
	static HRESULT STDMETHODCALLTYPE name params { return S_OK; }

SINK_STUB(sink_StartOperations, (IFileOperationProgressSink *t))
SINK_STUB(sink_FinishOperations, (IFileOperationProgressSink *t, HRESULT hr))
SINK_STUB(sink_PreRenameItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, LPCWSTR n))
SINK_STUB(sink_PostRenameItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, LPCWSTR n, HRESULT hr, IShellItem *c))
SINK_STUB(sink_PreMoveItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, IShellItem *d, LPCWSTR n))
SINK_STUB(sink_PostMoveItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, IShellItem *d, LPCWSTR n, HRESULT hr, IShellItem *c))
SINK_STUB(sink_PreCopyItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, IShellItem *d, LPCWSTR n))
SINK_STUB(sink_PostCopyItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *i, IShellItem *d, LPCWSTR n, HRESULT hr, IShellItem *c))
SINK_STUB(sink_PreNewItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *d, LPCWSTR n))
SINK_STUB(sink_PostNewItem, (IFileOperationProgressSink *t, DWORD f, IShellItem *d, LPCWSTR n, LPCWSTR tn, DWORD attr, HRESULT hr, IShellItem *c))
SINK_STUB(sink_UpdateProgress, (IFileOperationProgressSink *t, UINT total, UINT sofar))
SINK_STUB(sink_ResetTimer, (IFileOperationProgressSink *t))
SINK_STUB(sink_PauseTimer, (IFileOperationProgressSink *t))
SINK_STUB(sink_ResumeTimer, (IFileOperationProgressSink *t))

static IFileOperationProgressSinkVtbl sink_vtbl = {
	sink_QueryInterface, sink_AddRef, sink_Release,
	sink_StartOperations, sink_FinishOperations,
	sink_PreRenameItem, sink_PostRenameItem,
	sink_PreMoveItem, sink_PostMoveItem,
	sink_PreCopyItem, sink_PostCopyItem,
	sink_PreDeleteItem, sink_PostDeleteItem,
	sink_PreNewItem, sink_PostNewItem,
	sink_UpdateProgress, sink_ResetTimer, sink_PauseTimer, sink_ResumeTimer,
};

// trash_item sends the item named by the NUL-terminated UTF-16 path `wpath`
// to the Recycle Bin. It returns an HRESULT (>= 0 on success). *refused is
// set to 1 if the item cannot be recycled and was therefore left alone.
//
// Every COM object is released and COM is un-initialised on the same thread
// before returning (a cgo call stays on one OS thread), so nothing leaks into
// the Go program.
static long trash_item(const void *wpath, int *refused) {
	HRESULT hr_init, hr;
	IFileOperation *op = NULL;
	IShellItem *item = NULL;
	Sink *sink = NULL;
	DWORD cookie = 0;
	int advised = 0;
	BOOL aborted = FALSE;

	*refused = 0;

	hr_init = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE);
	if (FAILED(hr_init) && hr_init != RPC_E_CHANGED_MODE) {
		return hr_init;
	}
	// RPC_E_CHANGED_MODE: the thread already has COM in another mode. Use it
	// as is, and do not balance with CoUninitialize.

	hr = CoCreateInstance(&k_CLSID_FileOp, NULL, CLSCTX_ALL, &k_IID_FileOp, (void **)&op);
	if (FAILED(hr)) goto done;

	// FOF_NOCONFIRMATION/FOF_SILENT/FOF_NOERRORUI: never show UI.
	// FOF_WANTNUKEWARNING: if, despite the sink, the shell wanted to destroy
	// the item permanently, it must warn instead of doing it silently.
	hr = IFileOperation_SetOperationFlags(op,
		FOF_ALLOWUNDO | FOFX_RECYCLEONDELETE | FOF_NOCONFIRMATION |
		FOF_SILENT | FOF_NOERRORUI | FOF_WANTNUKEWARNING);
	if (FAILED(hr)) goto done;

	hr = SHCreateItemFromParsingName((PCWSTR)wpath, NULL, &k_IID_ShellItem, (void **)&item);
	if (FAILED(hr)) goto done;

	sink = (Sink *)calloc(1, sizeof(Sink));
	if (sink == NULL) {
		hr = E_OUTOFMEMORY;
		goto done;
	}
	sink->iface.lpVtbl = &sink_vtbl;
	sink->refs = 1; // our reference; Advise() adds its own
	sink->delete_hr = S_OK;

	hr = IFileOperation_Advise(op, &sink->iface, &cookie);
	if (FAILED(hr)) goto done;
	advised = 1;

	hr = IFileOperation_DeleteItem(op, item, NULL);
	if (FAILED(hr)) goto done;

	hr = IFileOperation_PerformOperations(op);
	if (sink->refused) {
		*refused = 1;
		hr = E_ABORT;
	} else if (SUCCEEDED(hr) && sink->saw_post && FAILED(sink->delete_hr)) {
		hr = sink->delete_hr;
	} else if (SUCCEEDED(hr)) {
		IFileOperation_GetAnyOperationsAborted(op, &aborted);
		if (aborted) {
			hr = E_ABORT;
		}
	}

done:
	if (advised)  IFileOperation_Unadvise(op, cookie);
	if (sink)     sink_Release(&sink->iface);
	if (item)     IShellItem_Release(item);
	if (op)       IFileOperation_Release(op);
	if (SUCCEEDED(hr_init)) CoUninitialize();
	return (long)hr;
}
*/
import "C"

import (
	"fmt"
	"io/fs"
	"syscall"
	"unsafe"
)

func newBackend() backend { return windowsBackend{} }

type windowsBackend struct{}

func (windowsBackend) move(abs string, _ fs.FileInfo) error {
	// A Go-allocated UTF-16 buffer holding no Go pointers may be handed to C
	// for the duration of the call; C neither keeps nor frees it.
	wpath, err := syscall.UTF16FromString(abs)
	if err != nil {
		return err
	}
	var refused C.int
	hr := int32(C.trash_item(unsafe.Pointer(&wpath[0]), &refused))
	if refused != 0 {
		return fmt.Errorf("%w: it would be deleted permanently instead of recycled", ErrNotTrashable)
	}
	return hresultError(hr)
}

// hresultError converts an HRESULT into a Go error (nil for success).
// HRESULTs wrapping a Win32 code become syscall.Errno, so that errors.Is
// works with fs.ErrNotExist and fs.ErrPermission.
func hresultError(hr int32) error {
	if hr >= 0 {
		return nil
	}
	u := uint32(hr)
	if u&0xFFFF0000 == 0x80070000 { // HRESULT_FROM_WIN32
		return syscall.Errno(u & 0xFFFF)
	}
	return hresult(u)
}

type hresult uint32

func (h hresult) Error() string { return fmt.Sprintf("recycle bin operation failed: HRESULT 0x%08X", uint32(h)) }
