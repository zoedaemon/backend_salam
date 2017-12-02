<?php
echo "Send to server";
error_reporting( E_ALL & ~E_NOTICE ^ E_DEPRECATED); 
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
	"di lokasi Sabaru terdapat lubang-lubang yg menganga <a href=\\\"localhost/script_evil.php\\\">klik diri q</a>' AND id LIKE %"
/*	
	"malam td rumah kami kebanjiran, dan sampai sekarang belum surut2, tolong kirim bantuan",
	"kebakaran di daerah jalan rusak flamboyan segera kirim pemadam kebakaran skrg",
	"Terjadi kerusakan jalan di sekitaran jalan rajawali palangkaraya",
	"Banjir di sekitaran daerah katingan",
	"kebakaran di daerah flamboyan segera kirim pemadam kebakaran skrg",
	"pembakar lahan wajib ditangkap segera !!!",
	"penumpukan sampah mengakibatkan banjir disepanjang jalan daerah kasongan",
	"bakar bakar itu sampah sampai asap kemana-mana",
	"Kliatan tuh jalan berlubang di skitaran jalan arah buntok",
	"lubang di jalan mana tuh harus ditutupin",
	"parah rusak parah tuh parit di dekat lampu merah tingang"*/
	);


$server = "localhost";
$username = "phpmyadmin";
$database = "salamdb";
$password = "adm19adm89";

// Koneksi dan memilih database di server
if (!NO_SQL_CHECK) {
	$mysqli = new mysqli($server,$username,$password, $database) ;
	if ($mysqli->connect_errno) {
	    echo "Failed to connect to MySQL: (" . $mysqli->connect_errno . ") " . $mysqli->connect_error;
	}
	echo $mysqli->host_info . "\n";
}
$lastID = array();

foreach ($messages as $msg ) {
	$check_id = 1;
	$ID  = rand(1,500);

	if (!NO_SQL_CHECK) {

		while ($check_id > 0  && !in_array($ID, $lastID)) {
			$ID  = rand(1,100);
			$sql = "SELECT id FROM pelaporan WHERE id = '$ID'";
			$qry = $mysqli->query($sql);

			$check_id = $qry->num_rows;
			//usleep(1500000);
			$qry->close();
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
