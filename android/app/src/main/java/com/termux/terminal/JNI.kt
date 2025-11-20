package com.termux.terminal

object JNI {
    init {
        System.loadLibrary("termux")
    }

    @JvmStatic
    external fun createSubprocess(
        cmd: String,
        cwd: String,
        args: Array<String>,
        envVars: Array<String>,
        processIdArray: IntArray,
        rows: Int,
        columns: Int,
        cell_width: Int,
        cell_height: Int
    ): Int

    @JvmStatic
    external fun setPtyWindowSize(fd: Int, rows: Int, cols: Int, cell_width: Int, cell_height: Int)

    @JvmStatic
    external fun waitFor(pid: Int): Int

    @JvmStatic
    external fun close(fileDescriptor: Int)
}
