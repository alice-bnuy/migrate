sudo systemctl enable bluetooth
sudo systemctl start bluetooth
bluetoothctl
power on
agent on
default-agent
scan on
// aguarde aparecer o teclado e pegue o MAC
pair XX:XX:XX:XX:XX:XX
// digite o PIN no teclado bluetooth e pressione Enter nele
connect XX:XX:XX:XX:XX:XX
trust XX:XX:XX:XX:XX:XX
scan off
exit
