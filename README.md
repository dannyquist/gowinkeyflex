# gowinkeyflex

Small utility to relay CW commands received by a K1EL WinKeyer and send to a FlexRadio Winkeyer COM port interface. 
Currently it is only a command line, a GUI will be forthcoming for the keyboard averse.

# Dependencies

* [Go 1.18](https://go.dev/dl/)
* [git command line tools](https://gitforwindows.org/)

All commands to be executed in PowerShell.

# Build

Start by cloning the repository:

```powershell
git clone https://github.com/dannyquist/gowinkeyflex.git
```

Get the depedencies:

```powershell
go get -v .
```

Build the executable

```powershell
go build .
```

# Running

Execute the command line with your configured serial ports. My K1EL WinKeyer is running on COM3 
and I have configured my FlexRadio to have a WinKeyer interface on COM11.

```powershell
gowinkeyflex.exe -k1el=COM3 -flex=COM11
```

If everything works, you should see text like this:

```
2022/07/07 22:19:18 Opening Flex serial port COM11
2022/07/07 22:19:18 Opening Winkeyer serial port COM3
2022/07/07 22:19:18 Flex Version: 10
2022/07/07 22:19:19 K1EL Version: 31
Speed change: 10
```

You may now use your paddle connected to the WinKeyer. Set SmartSDR to CW, and select an appropriate testing frequency. 
Any changes to the speed knob on the K1EL keyer will be relayed to the FlexRadio.

# Contributing

Pull requests are welcome!

# License

MIT License

Copyright (c) 2022 Danny Quist

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
