<?php
echo "Direct Reply SMS";

$secret = "2183781237693280uijshads";

$server = "localhost";
$username = "phpmyadmin";
$database = "gammu";
$password = "adm19adm89";

// Koneksi dan memilih database di server
$mysqli = new mysqli($server,$username,$password, $database) ;
if ($mysqli->connect_errno) {
    echo "Failed to connect to MySQL: (" . $mysqli->connect_errno . ") " . $mysqli->connect_error;
}
echo $mysqli->host_info . "\n";

/*
$sql = "SELECT ID, SenderNumber, TextDecoded FROM inbox WHERE processed = 'false'";
$qry = mysql_query($sql) or die(mysql_error());
 
while ($arr = mysql_fetch_array($qry)) {
       mysql_query("INSERT INTO outbox(DestinationNumber, TextDecoded) VALUES ('".$arr['SenderNumber']."', 
			'Halo, ini balasan sms anda. isi sms sebelumya\"".$arr['TextDecoded']."\"')") or die(mysql_error());
 
       mysql_query("UPDATE inbox SET processed='true' WHERE ID='".$arr['ID']."'");
}
*/

//mysql_query("INSERT INTO outbox(DestinationNumber, TextDecoded) VALUES ('082297335657','Balasan Otomatis dari Gammu')") or die(mysql_error());

$sql = "SELECT ID, SenderNumber, TextDecoded FROM inbox WHERE processed = 'false' ORDER BY ReceivingDateTime DESC";
$qry = $mysqli->query($sql);

 
if ($arr = $qry->fetch_array(MYSQLI_BOTH)) {
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
		//NOTE : gunakan table "queue_failed_sms" mysql yang dah dibuat  ini
	    echo 'OFFLINE: ' . socket_strerror(socket_last_error( $socket ));
	}

	$a = socket_write($socket, "$secret\n");
	var_dump($a);

	$a = socket_write($socket, '{"id":'.$arr['ID'].', "no-telp":"'.$arr['SenderNumber'].'", "sms":"'.$arr['TextDecoded']."\", \"secret\":\"$secret\"}\n");
	var_dump($a);

	$sql = "UPDATE inbox SET processed='true' WHERE ID='".$arr['ID']."'";
	$qry = $mysqli->query($sql);
	echo $qry->error;
}
?>
