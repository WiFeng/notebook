# MySQL  Innodb 架构

1. In-Memory Structures
2. On-Disk Structures

## In-Memory Structures

### 1.Buffer Pool

Buffer Pool 是innodb 缓存表数据及索引数据的主要内存空间，允许频繁访问的数据直接通过内存访问。在专用的服务器上，通常把物理内存的80%以上分派给buffer pool。

为了提高大容量读操作的效率，buffer pool 被分成了可以保存多行（mutiple row）的页（page）。为了缓存的有效管理，buffer pool 以 page 链表（linked list）的形式实现，使用一种类似LRU的实现，最近很少访问的页将从缓存中删除。

了解如何利用 buffer pool 使频繁访问的数据保持内存中，是Mysql 调优很重要的一方面。

提供的配置选项：

```config
innodb_buffer_pool_size 可以初始化时配置，也可以在线配置。
innodb_buffer_pool_chunk_size  默认128M，只能在初始化时配置。
innodb_buffer_pool_instances 实例个数：innodb_buffer_pool_size / innodb_buffer_pool_chunk_size
```

### 2. Change Buffer

Change Buffer 是一种特殊的数据结构，当二级索引不在buffer pool 中时，change buffer 将把二级索引页的变更进行缓存。这些变更包括 INSERT /UPDATE / DELETE 操作（DML），之后当其他的读操作把这个二级索引页加载到buffer pool 时，change buffer 中对应的页将会合并到 buffer pool 中。

与 buffer pool 相同，Change buffer也会在系统空闲或者是正常关闭时，purge 操作把更新的索引页（updated index page）写入磁盘。purge 操作把一系列索引修改按块写入要比每次修改立即写入更加高效。

如果有很多影响的行，或者大量的二级索引需要更新，change buffer 合并可能需要数小时。在这期间，磁盘I/O 会增加，可能导致依赖磁盘IO的查询会明显的变慢。

提供2个配置选项：

```config
innodb_change_buffering 允许哪些操作使用change buffer, 默认值：all（all/none/inserts/deletes/changes/purge）
innodb_change_buffer_max_size 允许的change buffer 的最大值。这个值是相对于 buffer pool 的百分比。默认值：25，最大值：50
```

### 3. Log Buffer

Log Buffer 缓存用户写入磁盘日志的文件内容。这个应该是只应用于 redo log ?

提供3个配置选项：

```config
innodb_log_buffer_size  Log Buffer 的大小， 默认值 16MB。当超过时，把内容写入到磁盘文件。较大的 size 值，可以让大型事务在commit 前，不需要把 redo log 写入文件。因此，如果需要插入、更新、删除很多记录时，增加这个size值，以减少磁盘 I/O开销。
innodb_flush_log_at_trx_commit  控制有多少个内容写入log buffer时，触发写入磁盘的操作。
innodb_flush__log_at_timeout 控制log buffer 刷入磁盘的频率。
```

## On-Disk Structures

### 1. Tablespace

#### a. System Tablespaces

system tablespaces 是一个存储区域，其中包括 InnoDB Data  Directory，doublewrite buffer, change buffer, undo logs。如果数据表被创建在 system tablespace 中，而不是 file-per-table 方式或是 general tablespaces 时，system tablespaces 中还包括这些数据表的数据及索引数据。

system tablespaces 可以有一个或者多个数据文件。初始化时，一个默认的数据文件 ibdata1 会在数据目录创建。文件大小与个数是在启动选项 innodb_data_file_path 来配置。

为了避免非常大的 system tablespaces ，对于用户表数据考虑使用 file-per-table tablespaces。同时，file-per-table tablespaces 也是默认的 tablespace 类型，当你创建表时会隐式使用。不像 system tablespaces , 在 file-per-table space 中，当对表执行truncate 或者 drop 时，磁盘空间将返回给操作系统。

提供的配置选项：

```config
innodb_data_file_path 数据文件名/初始大小/单次增长大小(如：ibdata1:10M:autoextend) ，要配置多个文件，需要使用分号隔开。
innodb_autoextend_increment 单次增长大小，默认 8M
```

#### b. File-Per-Table Tablespaces

File-per-table tablespace 为Innodb 表保存数据及索引数据，并且为每个表单独使用一个数据文件。这也是 InnoDB的默认行为，可以通过 innodb_file_per_table 变量控制。关闭这个选项将使得InnoDB在 system tablespace中创建表。

innodb_file_per_table 可以正配置文件中指定，也可以在运行时通过SET GLOBAL 语句配置。

file-per-table tablespace的优势：

* i: 当对使用 file-per-table 方式创建的表执行 truncate / drop 操作后，磁盘空间可以返还给操作系统。而对使用 shared 方式创建的表执行这些操作时，只会把空闲的空格交给shared 空间，这个空间后续只能被InnoDB使用。换言之，一个 shared tablespace 在执行truncate / drop 操作后，文件尺寸并不能减少。
* ii: 对file-per-table tablespace 中的表执行 truncate 性能更好。
* iii: file-per-table tablespace 的数据文件可以根据 IO 优化、空间管理、备份的目的创建在不同的存储设备。
* iv: 可以从另外一个mysql实例导入一个属于 file-per-table tablespace 的表。
* v: 在 file-per-table tablespace 创建的表使用了 Barracuda 文件格式，其中支持 Dynamic / compressed 特征。
* vi: 当发生数据损坏，当备份或者binlog不可用，或者当mysql 实例无法重启时，独立的tablespace 可以为成功的恢复节省时间与提供更大的可能性。
* vii: 可以使用 MySQL Enterprise Backup 为每个表执行不同的备份策略与计划。
* viii: 可以通过监控每个表的文件大小来监控数据表大小。
* ix: 通用的linux 文件系统不允许对一个数据文件的并发写入（inndb_flush_method=O_DIRECT）。因此，这种情况下使用 file-per-table tablespace 有可能性能得到改善。
* x: shared tablespace 的表由于 tablespace 最大尺寸 64TB 而受限。相比之下，每个 file-per-table tablespace 支持 64TB，为各个表的大小增长提供了大量的空间。

file-per-table tablespace 的劣势：

* i: 使用 file-per-table tablespace，每个表可能有一些不使用的空间，但是这些空间只能给相同表中的行使用，如果没有适当管理，可能造成空间的浪费。
* ii: 因为 fysnc在多个 file-per-table 数据文件上执行，而不是在单个的 shared tablespace ，因此会导致更高次数的fsync 操作。
* iii: mysqld 必须要针对每个file-per-table tablespace 保持一个打开文件的处理。如果你在 file-per-table tablespaces 有大量的表时可能影响性能。
* iv: 当每个表都有对应的文件时，就会有更多的文件描述符（file descriptor）
* v: 可能会有更多的碎片，而这影响 drop table 或者 table scan 的性能。然而，如果整理了碎片后，file-per-table tablespace 可以为这些操作改善性能。
* vi: 当删除一个在  file-per-table tablespace 中的数据表时，会扫描 buffer pool，对于较大的 buffer pool 可能会花费数秒钟。这个扫描会进行比较大的内部锁，将延迟其他事务。
* vii: 当使用 file-per-table tablespace 时，innodb_autoextend_increment 变量将不生效。

提供的配置选项：

```config
innodb_file_per_table=ON/OFF
```

#### c. General Tablespaces

General Tablespace 是一个共享的 InnoDB tablespce，是使用 CREATE TABLESPACE 语法创建。

General Tablespace 的功能：

* i: 与 system tablespace 类似，general tablespace 也是属于shared tablespaces，可以为多个表存储数据。
* ii: General tablespace 比 file-per-table tablespaces 具有可能的内存优势。在一个 tablespace的生命周期中，server一直将其元数据保存在内存中。general tablespace 中的多个表比分别在 file-per-table tablespace 中相同数据量的表，在元数据内存开销上更少。
* iii: general tablespace 可以放在与mysql data 目录相对位置，也可以放在独立的位置。提供了一种如 file-per-table tablespace 般对多个数据文件及存储的管理能力。正如 file-per-table tablespaces 一样, 可以把数据文件放在Mysql data 目录之外，允许你对特定的表分别进行性能管理、对特殊的表配置RAID/ DRBD、或者绑定table 到特殊的磁盘。
* iv: General tablespaces 支持  Antelope 与 Barracuda 两种文件格式。因此支持所有的 row format 及相关的特征。
* v: 使用 CREATE TABLE 创建表时，可以使用 TABLESPACE 选项来指定 tablespace 类型，如：general tablespace / file-per-table talespace / system tablespace
* vi: 在 ALTER TABLE  时，使用TABLESAPCE 选项，可以在不同的 tablespace 类型之间移动表。在这之前，从 file-per-table tablespace 移动到 system tablespace 是不可能的。自从有了 general tablespace ， 现在你可以这样做。

General Tablespace 的限制：

* i: 已经生成的、或者是已经存在的 tablespace 不能够转换为 general tablespace。
* ii: 创建临时的 general tablespace 是不支持的。
* iii: genral tablespace 不支持临时表。
* iv: 在gernal tablespace 中的数据表可能只能被支持 gernal tablespace 的Mysql 访问。
* v: 与system tablespace 类型，truncate /drop table 将不会释放磁盘空间，而是在 general tablespace 内部创建了可用的空闲空间，而这个空间只能被InnoDB使用。另外，针对 table-copying 方式的 ALTER TABLE 操作会增加对 tablespace 的使用，这些操作要求的空间是数据加索引的总和。因 table-copying 增加的空间，并不会释放给操作系统，而对于 file-per-table tablespace 中的表确可以做到。
* vi: 对于属于 general tablespace  的表，不支持 ALTER TABLE ... DISCARD TABLESPACE / ALTER TABLE ...  IMPORT TABLESPACE
* vii: 在Mysql 5.7.24 中对于general tablespace 存放表分区功能将被弃用，并且会在将来的Mysql版本中移除。
* viii: ADD DATAFILE 子句在同一个机器上执行 Master/Slave 复制的模式下是不支持的，因为它将导致 master / slave 在相同的位置创建相同名称的 tablespace 。

#### d. Temporary Tablespaces (ibtmp1)

Non-compressed 、用户定义的临时表以及基于磁盘的内部临时表都会创建在共享的(shared) temporary tablespace 中。配置选项 innodb_temp_data_file_path 为 temporary tablespace 的数据文件定义了相对路径、名称、大小。如果未设定，默认的行为时在 innodb_data_home_dir 目录创建自动扩展的，名称为 ibtmp1 的数据文件，略大于12MB。

在MySQL 5.6中，如果开启 innodb_file_per_table 选项， 非压缩的临时表将在 temporary 文件目录下以独立的 tablespace 形式创建。如果未开启 innodb_file_per_table 选项，则在 data 文件目录以 system tablespace 的形式创建。在Mysql 5.7 中，引入 shared temporary tablesapce 移除了针对每个 file-per-table tablespace 的临时表创建、移除的性能开销。专用的 temporary tablespace 意味着不需要保存临时表的元数据到Innodb 系统表中。

压缩的临时表（Compressed temporary tables），即创建表时使用 ROW_FORMAT=COMPRESSED 属性，将被创建在临时文件目录（temporary file directory） 中的 file-per-table tablespace 。

temporary tablespace 在正常的关闭，以及异常的初始化时被移除；每次服务启动的时候重新创建。当被重新创建时，temporary  tablespace 接收到一个动态生成的 space ID。如果 temporary tablespace 不能被创建，则将无法启动服务。如果服务异常停止，temporary tablespace 不会被移除。这种情况，数据看管理员可以手动移除 temporary tablespace 然后重启，这样会自动删除且重新创建 temporary tablespace。

temporary tablespace 不能直接放在 raw device 上。

默认，temporary tablespace 数据文件按需要扩展大小以容纳基于磁盘的临时表。与所有 shared tablespace 类似，如果某个临时表删除，释放的空间并不会返还给操作系统，而是返还给Innodb，可以在新的临时表需要时重新使用。

使用大型临时表、或者广泛使用临时表的环境，自动扩展的 temporary tablespace 数据文件可能会变得很大。基于临时表的长时间查询也会产生大数据文件。

为了阻止 temporary tablespace 数据文件变得太大，可以通过配置选项 innodb_temp_data_file_path 指定一个最大的文件大小。当文件到达指定的大小时，查询将会失败，表明数据表已满。配置  innodb_temp_data_file_path 需要重启服务。

另外，可以配置 default_tmp_storate_engine /  internal_tmp_disk_storage_engine，分别指定了用户创建的临时表与基于磁盘的内部临时表（on-disk internal temporary tables）使用的存储引擎。两个选项默认都是 InnoDB，MyISAM 存储引擎为每个临时表使用独立的文件，当临时表删除时，这个文件也对应的删除了。

#### e. Undo Tablespaces (undo_001/undo_002)

Undo logs 可以存储在一个或者多个 undo tablespace ，而不是必须要在 system tablespace。默认，undo logs 是保存在 system tablespace 中。基于 undo logs 的IO模式，把 undo log tablespace 放在SSD 存储上是一个好的选择，然后保持system  tablespace 依然在硬盘存储器上。

InnoDB 使用的 undo tablespace 数量是由配置选项 innodb_undo_tablespaces 控制。这个选项只能在Mysql初始化时配置，之后将不能修改。注意，这个配置选项被弃用 了，在将来的版本中将被删除。

在 undo tablespaces 中，undo tablespaces 与对应的 segments 将不能被删除，但是在其中的 undo logs 可以被 truncate。

提供的配置选项：

```config
innodb_undo_tablespaces
innodb_undo_directory
innodb_rollback_segments
innodb_undo_log_truncate
innodb_max_undo_log_size
```

### 2. DoubleWrite Buffer

当把 buffer pool 数据页 刷入到 Innodb 数据文件中对应位置前，会把对应的数据页先写入磁盘中的 DoubleWrite Buffer。当在写入输入页过程中，操作系统、存储子系统或者Mysqld进程崩溃时，Innodb 可以崩溃恢复中，在 DoubleWrite Buffer 中找到一个数据页很好的备份。

虽然数据写入了2次，但是并没有造成2倍的IO开销与2倍的IO操作。因为数据写入DoubleWrite Buffer 时以大量连续的块写入，但是只执行一次 fsync 系统调用。

在大多数场景下，doublewrite 默认是开启的。如果 system tablespace 文件（.ibdata）在支持原子写入的 Fusion-io 上时，doublewrite 自动关闭 。

### 3. Change Buffer

在内存结果中也有 change buffer，为什么磁盘结果中还有change buffer 呢？change buffer 不是会定期合并到 buffer pool ，或者是写入到对应的数据文件中吗？为什么还要在磁盘中单独存放 change buffer ?

在内存中change buffer 占据 buffer pool  的一部分空间。在磁盘中，change buffer 占据着系统表空间（system tablespace）的一部分。当Mysql 服务非正常关闭时，可以从这个磁盘的change buffer 中找到索引的改变。

虽然在异常情况发生时，通过上述的方式也不能完全避免数据的丢失，但是通过这个策略可以大幅降低对应数据丢失的可能。因为连续大块写磁盘的速度要比随机写快很多，花费时间越少，数据丢失的风险就越小，可以说MySQL 为了可靠性做了很多工作。

### 4. Undo Logs

Undo Log 是单个读写事务关联的撤销日志记录的集合。一个 undo log 记录中包含如何撤销聚簇索引记录上最后更改的信息。如果另一个开启了一致性读的事务需要查看原来的数据，未修改的数据将会从 undo log 中检索出来。undo log 是记录在 undo log segments 中，而 undo log segments 又包含在 rollback segments 中。Rollback segments 分别记录在 system tablespace、undo tablespace 和 temporary tablespace 。

在 temporary tablespace 中的 undo logs 用于对那些用户定义的临时表上执行的修改数据的事务。这些undo logs 不会记录到 redo log 中，因为他们不需要崩溃恢复。他们只是用于服务正常运行时的事务回滚。这种类型的 undo log 通过避免redo log I/O，使性能更好。

Undo logs 是保存在 rollback segments 中，Innodb 最大支持 128的 rollback segments，其中32个是为 temporary tablespaces 而申请的空间。这剩下的 96个  rollback segments 才可以分派给在普通表上的修改数据的事务。可以通过 innodb_rollback_segment 变量定义Innodb 使用的 rollback segments 个数。

一个 rollback segment 支持事务的个数取决于 rollback segments 中的 undo slot 个数 与 每个事务需要的 redo log 个数。rollback segments 中的 undo slot 个数因 innodb 的 page size 而不同。 undo slot 个数等于 page size / 16，比如 page size 等于 16KB 时，undo slot 个数是 1024。一个事务中可能分派到4种undo log，分别是 a.  在用户自定义表上的 INSERT b. 在用于自定义表上的 UPDATE / DELETE c. 用户自定义的临时表上的 INSERTR  d. 用户自定义的临时表上的 UPDATE/DELETE。Undo log 将按需分派，并不是所有的事务都分派这4种，可能只分派其中一种，也可能分派其中两种，取决于事务中实际的使用的 SQL类型 。

提供的配置选项：

```config
innodb_rollback_segment 最大值  128
```

### 5. Redo Log (ib_logfile0/ib_logfile1)

Redo Log 是一种基于文件的数据结构。用于在崩溃中恢复数据。在正常的操作中，redo log 记录改变表数据的请求，包括 SQL 语句与 low-level 的 api 调用。由于异常的关机，导致未完成的对数据文件的修改将会在初始化时自动重放。在初始化完成之前，连接是不允许的。
默认情况，redo log 物理上是指磁盘上的2个文件（ib_logfile0 / ib_logfile1）。mysql 以循环的方式写入 redo log。

InnoDB，如同其他支持ACID的数据库引擎，在事务提交之前，将把这个事务的redo log 刷入磁盘。InnoDB 使用group commit  功能，分组合并多个flush请求，避免一次 commit ,一个 flush。使用 group commit , InnoDB 把短时间内的多个用户事务通过一次写入文件执行提交，显著的改善了吞吐量。

提供的配置选项：

```config
innodb_log_file_size  单个redo文件的大小
innodb_log_file_ingroup  redo log 文件个数
```
