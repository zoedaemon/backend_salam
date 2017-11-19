<?php
echo "Send to server";

$socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
$connection =  @socket_connect($socket, '127.0.0.1', 1999);

if( $connection ){
    echo 'ONLINE';
}
else {
	//TODO: Jika golang server (BACKEND binary GoServer) mati (sebelum dihidupkan kembali oleh Supervisor), 
	//		yg harus dilakukan adalah simpan sms ke table antrian yang belum diproses; usahakan bisa satu servis 
	//		dengan BACKEND binary GoServer (goroutine terpisah) yang bisa mengecek table antrian tsb, jd gak perlu 
	//		servis lain untuk mengecek secar berkala data yg masuk ke table antrian

    echo 'OFFLINE: ' . socket_strerror(socket_last_error( $socket ));
}

/*
Warning: socket_write(): unable to write to socket [0]: An established connection was aborted by the software in your host machine.
*/
$messages = array(
	"Terjadi kerusakan jalan di sekitaran jalan rajawali palangkaraya",
	"Banjir di sekitaran daerah katingan",
	"malam td rumah kami kebanjiran, dan sampai sekarang belum surut2, tolong kirim bantuan",
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"pembakar lahan wajib ditangkap segera !!!",
	"penumpukan sampah mengakibatkan banjir disepanjang jalan daerah kasongan",
	"bakar bakar itu sampah sampai asap kemana-mana",
	"Terjadi kerusakan jalan di sekitaran jalan rajawali palangkaraya",
	"Banjir di sekitaran daerah katingan",
	"malam td rumah kami kebanjiran, dan sampai sekarang belum surut2, tolong kirim bantuan",
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"pembakar lahan wajib ditangkap segera !!!",
	"penumpukan sampah mengakibatkan banjir disepanjang jalan daerah kasongan",
	"bakar bakar itu sampah sampai asap kemana-mana",
	"Terjadi kerusakan jalan di sekitaran jalan rajawali palangkaraya",
	"Banjir di sekitaran daerah katingan",
	"malam td rumah kami kebanjiran, dan sampai sekarang belum surut2, tolong kirim bantuan",
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"pembakar lahan wajib ditangkap segera !!!",
	"penumpukan sampah mengakibatkan banjir disepanjang jalan daerah kasongan",
	"bakar bakar itu sampah sampai asap kemana-mana",

	);

foreach ($messages as $msg ) {
	$a = socket_write($socket, '{"no-telp":"082297335657", "sms":"'.$msg."\", \"secret\":\"2183781237693280uijshadj^^^^ds\"}\n");
	echo $msg."\n";
	var_dump($a);
}


?>
