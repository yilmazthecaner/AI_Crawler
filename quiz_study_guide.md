# Quiz Hazırlık Rehberi: SpiderSearch Teknik Altyapı

Pazartesi günkü quiz için bu rehber, projenin **"neden"** ve **"nasıl"** yapıldığını en ince detayına kadar açıklar.

---

## 1. Projenin Amacı (The Goal)
Bu projenin temel amacı, dış kütüphanelere (Scrapy, BeautifulSoup vb.) bağımlı kalmadan, bir programlama dilinin (Go) **çekirdek yeteneklerini** kullanarak ölçeklenebilir bir sistem tasarlamaktır.
- **Dinamik Tarama:** İnterneti bir ağ (graph) gibi görüp düğümler (URL) arasında gezinmek.
- **Gerçek Zamanlı İndeksleme:** Veri toplanırken aynı anda arama yapılabilmesini sağlamak.
- **Eşzamanlılık (Concurrency):** Tek bir makinede yüzlerce işlemi aynı anda güvenli bir şekilde yönetmek.

---

## 2. Teknik Bileşenler (Technical Components)

### A. Crawler (Örümcek/Tarayıcı) - *internal/crawler/crawler.go*
Sistem bir **Recursive (Özyinelemeli)** yapıdadır.
- **Depth (k):** Başlangıç sayfasından kaç "hop" uzağa gidileceğini belirler. (Depth 0: Başlangıç, Depth 1: Başlangıçtaki linkler...)
- **Visited Set (`sync.Map`):** Bir sayfayı iki kez taramamak için kullanılır. Eğer bu olmasaydı, döngüsel linkler (A -> B -> A) sistemi sonsuz döngüye sokardı.
- **Link Extraction:** `regexp` (Regular Expressions) kullanılarak HTML içindeki `href` etiketleri ayıklanır.

### B. Indexer (Dizin Oluşturucu) - *internal/index/index.go*
Veriler **Inverted Index (Ters Dizin)** yapısında saklanır.
- **Yapı:** `map[keyword][]ResultTriple`.
- Anahtar kelimeyi verdiğinizde, o kelimenin geçtiği tüm URL'lerin listesini saniyeler içinde döner.
- **Thread-Safety:** `sync.RWMutex` kullanılır. `RLock` (Okuma kilidi) birden fazla kişinin aynı anda arama yapmasına izin verirken, `Lock` (Yazma kilidi) tarayıcının yeni veri eklerken veriyi bozmamasını sağlar.

### C. Searcher (Arama Motoru) - *internal/searcher/searcher.go*
- **Heuristic Ranking (Sıralama Algoritması):** Arama sonuçlarını neye göre sıralıyoruz?
  - Kelimenin sayfa içindeki **frekansı** (kaç kez geçtiği).
  - Kelimenin **Title (Başlık)** içinde geçip geçmediği (Başlıkta geçiyorsa +10 puan gibi bir ağırlık ekliyoruz).

---

## 3. Kritik Konseptler (Must-Know Concepts)

### 1. Back-Pressure (Geri Basınç / Yük Yönetimi)
Sistem internetteki milyonlarca sayfaya aynı anda saldırmamalıdır, yoksa hem bizim RAM'imiz biter hem de karşı site bizi engeller.
- **Çözüm:** `semaphore` (kanal üzerinden kısıtlama).
- Kodda `make(chan struct{}, maxWorkers)` şeklinde bir kanal oluşturduk. Her yeni "işçi" (worker) bu kanala bir jeton atar, işi bitince geri alır. Kanal doluysa yeni işçi bekler.

### 2. Goroutines & WaitGroups
- **Goroutine:** Go'nun hafif iş parçacıklarıdır. Binlerce sayfayı aynı anda taramamızı sağlar.
- **sync.WaitGroup:** Tüm tarama işlemi bitene kadar ana programın kapanmamasını sağlar. `Add(1)` ile başlatırız, her iş bitince `Done()` deriz.

### 3. Networking (`net/http`)
Dış kütüphane yerine Go'nun standart `http.Get` metodunu kullanıyoruz. Gelen yanıtın (Response) gövdesini `io.ReadAll` ile okuyup metin (string) haline getiriyoruz.

---

## 4. Akış Özeti (The Workflow)
1. `main.go` kullanıcıdan parametreleri alır.
2. `Crawler` başlatılır ve ilk URL'yi (`origin`) sıraya koyar.
3. Her URL için bir `Goroutine` açılır (Semaphore izin verdiği sürece).
4. Sayfa indirilir -> Linkler ayıklanır -> Kelimeler dizine eklenir.
5. Aynı anda `Web UI` (port 8080) ayağa kalkar ve kullanıcıdan gelen sorguları bekler.

---

## Quize Hazırlık Soruları (Self-Test)
1. **Soru:** Neden `sync.Map` veya `Mutex` kullanıyoruz?
   - **Cevap:** Aynı anda çalışan yüzlerce Goroutine aynı bellek bölgesine (map) veri yazmaya çalışırsa "Race Condition" oluşur ve program çöker. Mutex bu trafiği yönetir.
2. **Soru:** Projedeki "Back-pressure" nasıl sağlanıyor?
   - **Cevap:** `chan struct{}` tipinde bir kanal (semaphore) kullanılarak; aktif çalışan işçi sayısı `maxWorkers` değerini aşamaz.
## 5. Brightwave Spesifik Altyapı (Quiz'de Çıkabilir!)

### A. Sharded Filesystem Storage
Büyük veri setlerini tek bir dosyada tutmak yerine, kelimelerin baş harflerine göre bölüyoruz (`storage/a.data`, `storage/b.data` vb.).
- **Avantajı:** Arama yaparken tüm dizini değil, sadece ilgili harfin dosyasını okuyarak hızı artırırız.

### B. Job Identification
Her tarama işlemi `[Epoch]_[ID]` formatında benzersiz bir kimlik alır.
- **Örnek:** `1710850000_123.data` dosyası o işin tüm loglarını ve durumunu JSON formatında saklar.

### C. curl Kullanımı
Networking için Go'nun iç kütüphanesi yerine sistemdeki `curl` komutu `os/exec` ile çağrılır. Bu, sistem seviyesinde işlemler yapabildiğimizi gösterir.

### D. Long Polling
Status sayfasında sayfa yenilenmeden verilerin güncellenmesi için "HTTP Long Polling" tekniği kullanılır. Sunucu, veri hazır olana kadar bağlantıyı açık tutmaya çalışır.
