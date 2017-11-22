<?php
echo "Send to server";
define(NO_SQL_CHECK, FALSE);
$secret = "2183781237693280uijshads";

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
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"Terjadi kerusakan jalan di sekitaran jalan rajawali palangkaraya",
	"Banjir di sekitaran daerah katingan",
	"malam td rumah kami kebanjiran, dan sampai sekarang belum surut2, tolong kirim bantuan",
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"pembakar lahan wajib ditangkap segera !!!",
	"penumpukan sampah mengakibatkan banjir disepanjang jalan daerah kasongan",
	"bakar bakar itu sampah sampai asap kemana-mana",
	"Kliatan tuh jalan berlubang di skitaran jalan arah buntok",
	"lubang di jalan mana tuh harus ditutupin",
	"parah rusak parah tuh parit di dekat lampu merah tingang"
	);


$server = "localhost";
$username = "root";
$database = "salamdb";
$password = "";

// Koneksi dan memilih database di server
if (!NO_SQL_CHECK) {
	mysql_connect($server,$username,$password) or die("Koneksi gagal");
	mysql_select_db($database) or die("Database tidak bisa dibuka");
}
$lastID = array();

foreach ($messages as $msg ) {
	$check_id = 1;
	$ID  = rand(1,100);

	if (!NO_SQL_CHECK) {

		while ($check_id > 0  && !in_array($ID, $lastID)) {
			$ID  = rand(1,100);
			$sql = "SELECT id FROM pelaporan  WHERE id = '$ID'";
			$qry = mysql_query($sql);
			$check_id = mysql_num_rows($qry);
			//usleep(1500000);

			var_dump($ID);
		}
	}

	$lastID[] = $ID;
	var_dump($lastID);
	//send kode keamanan dulu ya hehe
	$a = socket_write($socket, "$secret\n");
	var_dump($a);
	$a = socket_write($socket, '{"id":'.$ID.', "no-telp":"082297335657", "sms":"'.$msg."\", \"secret\":\"2183781237693280uijshads\"}\n");
	echo $msg."\n";
	var_dump($a);
}


?>
