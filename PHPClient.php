<?php
echo "Direct Reply SMS";


$server = "localhost";
$username = "root";
$database = "gammu";
$password = "";

// Koneksi dan memilih database di server
mysql_connect($server,$username,$password) or die("Koneksi gagal");
mysql_select_db($database) or die("Database tidak bisa dibuka");

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
$qry = mysql_query($sql) or die(mysql_error());
 
if ($arr = mysql_fetch_array($qry)) {
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
	$a = socket_write($socket, '{"no-telp":"'.$arr['SenderNumber'].'", "sms":"'.$arr['TextDecoded'].'", "secret":"2183781237693280uijshadj%%$ds"}');
	var_dump($a);

	mysql_query("UPDATE inbox SET processed='true' WHERE ID='".$arr['ID']."'");
	echo mysql_error();
}
?>