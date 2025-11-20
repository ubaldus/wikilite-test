package it.eja.wikilite

import android.app.Dialog
import android.os.Bundle
import android.view.View
import android.widget.ArrayAdapter
import android.widget.Spinner
import android.widget.TextView
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.DialogFragment

class DownloadDialog(
    private val filePath: String,
    private val listener: (String, String) -> Unit
) : DialogFragment() {

    private var selectedPath: String = ""

    override fun onCreateDialog(savedInstanceState: Bundle?): Dialog {
        val inflater = requireActivity().layoutInflater
        val view = inflater.inflate(R.layout.dialog_download, null)

        val tvFileInfo: TextView = view.findViewById(R.id.tvFileInfo)
        val spinnerLocation: Spinner = view.findViewById(R.id.spinnerLocation)

        val fileName = filePath.substringAfterLast("/")
        tvFileInfo.text = "Download: $fileName"

        val (locations, paths) = getAvailableStorageLocations()

        val adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_item, locations)
        adapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
        spinnerLocation.adapter = adapter

        spinnerLocation.onItemSelectedListener = object : android.widget.AdapterView.OnItemSelectedListener {
            override fun onItemSelected(parent: android.widget.AdapterView<*>?, view: View?, position: Int, id: Long) {
                selectedPath = paths[position]
            }

            override fun onNothingSelected(parent: android.widget.AdapterView<*>?) {
                if (paths.isNotEmpty()) {
                    selectedPath = paths[0]
                }
            }
        }

        if (paths.isNotEmpty()) {
            selectedPath = paths[0]
        }

        return AlertDialog.Builder(requireContext())
            .setView(view)
            .setTitle("Choose Download Location")
            .setPositiveButton("Download") { dialog, _ ->
                listener(filePath, selectedPath)
                dialog.dismiss()
            }
            .setNegativeButton("Cancel") { dialog, _ ->
                dialog.dismiss()
            }
            .create()
    }

    private fun getAvailableStorageLocations(): Pair<List<String>, List<String>> {
        val locationNames = mutableListOf<String>()
        val locationPaths = mutableListOf<String>()
        val storageDirs = requireContext().getExternalFilesDirs(null).filterNotNull()

        if (storageDirs.isNotEmpty()) {
            locationNames.add("Internal App Storage")
            locationPaths.add(storageDirs[0].absolutePath)
        }

        if (storageDirs.size > 1) {
            for (i in 1 until storageDirs.size) {
                locationNames.add("External App Storage")
                locationPaths.add(storageDirs[i].absolutePath)
            }
        }

        if (locationPaths.isEmpty()) {
            locationNames.add("Internal App Storage")
            locationPaths.add(requireContext().filesDir.absolutePath)
        }

        return Pair(locationNames, locationPaths)
    }
}