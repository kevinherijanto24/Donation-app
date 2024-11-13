Video Tutorial Demo penggunaan aplikasi: https://youtu.be/_2vQIM1oZ9A

Step untuk menjalankan program

1. CD ke directory dari folder ini dan panggil go run main.go di console. Server Websocket, TCP, dan UDP akan berjalan secara otomatis dan concurrent.
2. Bisa ke browser apapun, dan masukkan localhost:8082 untuk masuk ke tampilan webnya, akan diperlukan untuk memasukkan nama yang sekaligus membuat akun di server websocket.
3. Untuk melakukan deposit/withdraw bisa buat terminal baru dan CD ke folder Client. Setelah itu bisa menggunakan aksi go run client.go -deposit atau go run client.go -withdraw.
4. Dapat dilihat di tampilan web, saldo akan otomatis terupdate sesuai dengan yang dikirimkan.

Informasi Penggunaan Server:

Kirim Donasi antar orang - Websocket
Deposit Saldo - TCP
Withdraw Saldo - UDP
