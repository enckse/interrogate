const electron = require('electron')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const {session} = require('electron')
const path = require('path')
const url = require('url')

let mainWindow

function createWindow () {
    /*
  const filter = {
      urls: ['http://localhost:8080/']
  }
  session.defaultSession.webRequest.onBeforeRequest(filter, (details, callback) => {
  })*/

  mainWindow = new BrowserWindow({width: 800, height: 600})
  mainWindow.webContents.openDevTools()
  mainWindow.loadURL('http://localhost:8080')
  mainWindow.on('closed', function () {
    mainWindow = null
  })
}

app.on('ready', createWindow)

app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('activate', function () {
  if (mainWindow === null) {
    createWindow()
  }
})

