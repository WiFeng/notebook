# MySQL InnoDB 事务与锁

## InnoDB 锁

### a. 共享锁（shared Locks）、互斥锁（Exclusive Locks）

InnoDB 实现了标准的行级锁（row-level locking），包括2种类型，共享锁、互斥锁。

* 共享锁允许持锁的事务对行进行读取
* 互斥锁允许持锁的事务对行进行更新与删除

如果事务 T1 在行 r 上持有一个共享锁（S），那么一些不同的事务T2要在行 r 上持锁的处理如下：

* T2 想要持有 S 锁，则可以立即获得。结果，T1 与 T2 在行 r 上都获得了 S 锁。
* T2 想要持有 X 锁，则不能立即获得。
  
如果事务T1 在行 r 上持有一个互斥锁（X），那么一些不同的事务T2想要在行 r 上获得任何类型的锁都不能立即获得。然而，事务T2必须等待事务 T1 释放在行 r 上的锁。

### b. 意图锁（Intention Locks）

InnoDB 支持多种粒度的锁，允许行锁（row locks）与表锁（table locks）共存。例如，LOCK TABLE ... WRITE 声明在特定的表上获得一个互斥锁（exclusive lock / X lock）。为了在多个级别上实现锁，InnoDB 使用了意图锁（Intention Locks）。意图锁是表级锁（table-level locks），指出一个事务稍后将要在表中行上获得哪种类型的锁（共享锁或者互斥锁）。有2种类型的意图锁：

* 意图共享锁（Intention shared lock / IS）指出一个事务想要在表中单个行上设定共享锁（shared lock）
* 意图互斥锁（Intention exclusive lock / IX）指出一个事务想要在表中单个行上设置互斥锁（exclusive lock）

例如，SELECT ... LOCK IN SHARE MODE 设定了一个 IS 锁， SELECT ... FOR UPDATE 设定了一个 IX 锁。

意图锁协议如下：

* 一个事务在表中行上获得一个共享锁之前，它必须要首先获得一个在表上的 IS 锁或者更强的锁。
* 一个事务在表中行上获得一个互斥锁之前，它必须要首先获得一个在表上的 IX 锁。

表级锁类型兼容性盖如如下：
x
x

如果与一个锁与已经存在的锁是兼容的（compatible）， 则这个锁被授予给请求的事务，但是如果是冲突的（conflict）那么则不会。事务将等待直到与已经存在的锁冲突解除。如果一个锁请求与已经存在的锁由于即将导致死锁（deadlock）而冲突，则会产生一个错误。

意图锁不会阻塞除全表请求 full table requests（例如， LOCK TABLES ... WRITE）之外的的任何东西。意图锁的主要目的是展示有人正在锁定表中的一个行，或者是将要锁定表中一个行。

### c. 记录锁（Record Locks）

记录锁是指在索引记录上的锁。例如，SELECT c1 FROM t WHERE c1 = 10 FOR UPDATE; 阻止任何其他事务插入、更新、删除 t.c1 等于 10 的行。

记录锁总是对索引记录加锁，即使一个表没有定义任何索引。在这种情况下，InnoDB 创建一个隐藏的聚簇索引，使用这个索引进行记录锁。

### d. 间隙锁（Gap Locks）

间隙锁是索引记录之间、第一个索引记录之前、最后一个索引记录之后，这些间隙中的一种锁。例如，SELECT c1 FROM t WHERE c1 BETWEEN 10 AND 20 FOR UPDATE; 无论在该列上是否存在此值，都将阻止其他事务插入 t.c1  等于 15 的值，因为在这个范围里已经存在值的间隙都被加锁了。

一个间隙可能跨单个索引值，多个索引值，或者甚至为空。

间隙锁是性能与并发之前权衡的一部分，使用在部分事物隔离级别中，而不是全部隔离级别。

使用唯一索引查找唯一的行进行加锁时，是不需要间隙锁的。（这并不包括查找条件中包括多列唯一索引的其中一部分的情况，这种情况下，间隙锁依然是会发生的。）例如，如果 id 列有一个唯一的索引，这下面的声明只在 id 等于100 的行上加索引记录锁，并不管其他会话是否在之前的间隙插入行。

SELECT * FROM child WHERE id = 100;

如果 id 列没有索引，或者没有一个唯一索引，这个声明则对之前的间隙加锁。

这里还值得注意的是，不同的事务可以在一个间隙上持有冲突锁。例如，当事务B持有一个互斥锁（gap X-lock）时，事务A也可以在相同的间隙持一个有个共享的间隙锁（gap S-lock）。允许冲突的间隙锁，是因为如果一个索引上的记录被删除，不同事务在这个记录上持有的间隙锁必须要合并。

在InnoDB 中，间隙锁是纯抑制性的，他们唯一的目的是阻止其他事务插入记录到间隙中。间隙可以共存。一个事务持有一个间隙锁并不会阻止其他事务在相同的间隙持有间隙锁。他们彼此间不冲突，他们执行相同的函数。

间隙锁可以被明确地关闭。如果你改变事务级别为 READ_COMMITTED 或者是开启 innodb_lock_unsafe_for_binlog （现在已经弃用）系统变量，则意味着关闭间隙锁。在这些情况下，对于查找与索引扫描，则间隙锁是关闭的，只用于外键约束检测与重复键检测。

使用 READ_COMMITTED 隔离级别与开启 innodb_locks_unsafe_for_binlog 还有其他影响。在MySQL 评估了 WHERE 条件之后，没有匹配到的行上的记录锁则被释放。对于 UPDATE 声明，InnoDB使用半一致性读，它返回最新提交的版本，以便MySQL 可以决定是否对应的行匹配UPDATE 的 WHERE 条件。

### e. Next-key Locks

Next-key 锁是索引记录上的记录锁与索引记录之前间隙的间隙锁的结合。

当 InnoDB 搜索或者是扫描一个表的索引时，它在遇到的索引记录上设置共享锁或互斥锁。因此，行记录的锁实际上是索引记录锁。一个索引记录上的 Next-key 锁也包括这个索引记录之前的间隙。也就是，一个 next-key 锁是一个索引记录锁加这个索引记录之前间隙上的间隙锁。如果一个会话持有一个索引上记录 R 的共享锁或者是互斥锁，那么另一个会话则不能在按照这个索引顺序 R 之前立即插入一个新的索引记录。

假设，一个索引包含这么几个值， 10、11、13、20。针对这个索引可能的next-key 锁包含以下这些间隔，圆括号表示不包括间隔的边界值，方括号表示包括这个边界值。
(negative infinity, 10]
(10, 11]
(11, 13]
(13, 20]
(20, positive infinity)

对于最后的一个间隙，next-key 锁针对大于索引最大值与 supremum 记录之前的间隙加锁。supermum 这个记录是指他的值大于索引中真实存在的任何值。 supremum 记录并不是一个真正的索引记录，因此实际上，这个 next-key 锁对索引中最大值之后的间隙加锁。

默认， InnoDB 使用 READPEATABLE READ 事务隔离级别。在这种情况下，InnoDB 使用 next-key 锁进行查找与索引扫描，组织幻读行出现。

### f. 入意图锁（Insert Intention Locks）

插入意图锁是一个间隙锁类型，在行插入之前由 INSERT 操作设置。这个锁表示一种插入意图，如果多个事务在相同的间隙不是插入到相同的位置，则不需要彼此等待。假设，有2个索引记录 4、7。不同的事务分别试图插入 5、6，虽然每个事务在获取插入行上的互斥锁之前，都以插入意图锁的形式锁住了 4-7之前的间隙，但是并不会互相阻塞，因为这些行是不冲突的。

下面的例子描绘了在获取插入的记录上的互斥锁之前，一个事务将先获得插入意图锁。例子牵涉2个客户端， A与B。

客户端 A 创建了1个表，包含索引记录（90、102），然后开始一个事务，在ID 大于 100 的索引记录上放置一个互斥锁。这个互斥锁包含一个 102 记录之前的间隙锁。

CREATE TABLE
...

客户端B启动一个事务，往这个间隙中插入记录。当这个事务等获取一个互斥锁时，先获取到一个插入意图锁。

### g. 自增锁（AUTO-INC Locks）

自增锁是一个特殊的表级锁，当事务插入到一个有 AUTO_INCREMENT 列的表时使用。在最简单的情况下，如果一个事务正在往表里插入数据，任何其他插入该表的事务必须等待，以便插入记录的第一个事务可以接收到连续的主键值。

innodb_autoinc_lock_mode 配置选项控制了自增锁使用的算法。它允许你选择在自增值的可预测序列与插入操作的最大并发性之前进行权衡。

### h. 针对空间索引的预测锁（Predicate Locks for Spatial Indexes）

InnoDB 支持包含空间列的空间索引。

为了处理牵涉空间索引操作的锁，next-key 锁并不能很好的支持 REPEATEABLE READ 与 SERIALIZABLE 事务隔离级别。在多维数据中，并没有一个绝对排序的概念，所以不清楚哪个键是“下一个”键。

为了支持空间索引表的隔离级别，Innodb 使用预测锁。一个空间索引包含最小边界区域（MBR）值，因此InnoDB 针对查询通过在MBR值上设定预测锁来强制在所有上的一致性读。其他事务则不能插入、修改匹配到这个查询条件的行。

## 事务模式

### a. 事务隔离级别

事务隔离是数据库处理的基础。隔离是 ACID 中的 I ；通过隔离级别的设置，我们可以在性能与可靠性之间平衡，还可以调整一致性，以及当多个事务同时在执行更新与查询时对于结果的重复性。

InnoDB 总共提供4种事务隔离级别：READ_UNCOMMITED, READ_COMMITED, REPEATED_READ 与 SERIALIZABLE。对于 InnoDB ，默认的隔离级别是 REPEATABLE READ ，即可重复读。

用户可以通过 SET TRANSACTION 声明来修改单个会话的隔离级别。如果要为所有的连接设置默认的隔离级别，可通过启动命令行选项 --transaction-isolation 或者是配置文件来进行配置。

InnoDB 支持的每种事务隔离级别使用不同的 锁策略。针对需要完全符合ACID的重要数据，可以使用默认的“可重复读”级别强制一个高的一致性。在这些情形下，如大块上报（bulk reporting），精确的一致性与可重复的结果相对于减少因锁而造成的负载来说并不是那么重要，你可以使用“读提交”，甚至是“读未提交”来减缓一致性规则。“序列化”强制比“重复读”更严格的规则，主要使用在一些特殊的场景，诸如，使用了XA事务以及排查并发与死锁的问题。

下面列表描述了MySQL如何支持不同的事务级别。列表顺序是从最长使用的级别到最少。

### i. 可重复读

这是InnoDB 的默认隔离级别。在相同的事务中一致性读是指读有第一次读产生的快照。这意味着，如果你在相同的事务中提交了几个非阻塞的 SELECT 声明，这些 SELECT 语句彼此之间也是一致的。

对于加锁的读（SELECT .. FOR UPDATE 或者是 LOCK IN SHARE MODE），更新、删除语句，锁取决于是否使用了唯一索引及唯一查找条件，还是一个范围类型的查找条件。

对于使用唯一查找条件的唯一索引，InnoDB 只锁查找到的记录，而不会锁其之前的间隙。

对于其他搜索条件，InnoDB 对被扫描的所有范围加锁，使用间隙锁、或 next-key 锁阻塞其他事务插入范围内的间隙。

#### ii. 读提交

每个一致性读，甚至在相同的事务中，更新、读取自己的最新快照。

对于加锁的读（SELECT .. FOR UPDATE 或者是 LOCK IN SHARE MODE）、更新、删除语句，InnoDB 只对索引记录加锁，而不对他们之前的间隙加锁，因此允许在加锁的记录之后自由插入新记录。间隙锁只使用于外键约束检测以及重复键检测。

因为间隙锁是关闭的，其他事务可能在间隙中插入新记录，所以幻读问题可能发生。

在“读提交”隔离级别中，只支持 row-based 格式的 binlog。如果你在“读提交”中使用 binlog_format=MIXED, 则mysql 自动使用 row-based 格式。

使用“读提交”还有另外的作用：

对于 UPDATE / DELETE 语句，InnoDB 只针对它更新、删除的记录加锁。在MySQL 评估 where 条件之后，不匹配的记录锁将被释放。这极大的减少了死锁发生的可能，但是依然可能发生。

对于 UPDATE 语句，如果一个行已经被加锁，InnoDB 执行一个“半一致性”读，返回最新提交到MySQL的版本，以便MySQL可以决定是否这个行与 UPDATE 的 WHERE 条件匹配。如果匹配，则MySQL 再次读这行，这次 InnoDB 要么锁住它，要么等待在其行上加锁。

考虑如下的示例，表初始化信息如下：
...
...

在这个示例中，表没有索引，因此查找与索引扫描，加锁时使用隐藏的聚簇索引，而不是索引的列。

假设，一个会话使用这些语句执行了一个UPDATE:

.....
.....

假设，另一个会话在之后执行了如下的UPDATE:
......
......

当 InnoDB 执行每个UPDATE时，它首先对于读取到的所有行获取一个互斥锁，然后决定是否修改它，如果InnoDB 不修改这行，他将释放这个行上的锁。否则，InnoDB 则保持这个锁直到事务结束。这影响之后的事务处理。
当使用默认的“可重复读”隔离级别时，第一个UPDATE在读取到的每一行上都取得互斥锁，并且不会释放他们。

.....
.....

第二个UPDATE 尝试获取任何锁都会理解阻塞（因为第一个UPDATE在所有行上获得了锁），并且不会被处理直到第一个UPDATE提交或者是回滚。

如果使用“读提交”，第一个UPDATE在读取到的所有行上获得互斥锁，同时释放它不修改的行上的锁。

.....
.....

对于第二个 UPDATE，InnoDB 执行一个“半一致性”读，返回读取到各行的最新提交的版本给MySQL，以便MySQL可以决定是否这些行匹配UPDATE的WHERE条件。

....
....

然而，如果 WHERE 条件包含一个索引的列，并且InnoDB 使用了这个索引，那么获取、保留记录锁时只考虑索引的列。在下面的示例中，第一个UPDATE 获取并保留了 where b=2 条件下所有行上的互斥锁。第二个 UPDATE 试图在相同记录上获取互斥锁时阻塞，因为它也是使用在列b上的定义的索引。

....
....

使用“读提交”隔离级别与开启 innodb_locks_unsafe_for_binlog （已弃用）配置选项，除以下2点不同之外，其他影响都是相同的。

* 开启 innodb_locks_unsafe_for_binlog 是一个全局的设置项，影响所有会话，然而隔离级别可以针对所有会话全局设置，也可以为每个会话单独设置。
* innodb_locks_unsafe_for_binlog 只能在mysql 启动时设置，而隔离级别可以即可以在启动时设置，可以在运行时改变。

因而，“读提交”相比 innodb_locks_unsafe_for_binlog 可以提供更好、更灵活的控制。

#### iii. 读未提交

SELECT 语句以非锁方式执行，但是可能使用更早版本的行。因此，如果使用这种隔离级别，读是不一致的，这称之为“脏读”。否则，这种隔离级别像“读提交”一般工作。

#### iv. 序列化

这种级别类似于“可重复读”，但是如果 autocommit 关闭时，InooDB 隐式地转换锁的 SELECT 语句为 SELECT ... LOCK IN SHARE MODE 。如果 autocommit 开启时，SELECT 就是它自己的事务。因此，我们知道他是只读的，如果以一致的（非锁定的）读取方式执行，并且不需要为其他事务阻塞，则可以序列化它。（如果其他事务修改了所选的行，要强制使用普通的SELECT来阻塞，请禁用自动提交）。

### b. 自动提交，提交，回滚

在 InnoDB， 所有用户活动都发生在事务中。如果 autocommit 模式是开启的，每个SQL语句即为一个单独的事务。默认，MySQL 为每个新连接启动session时，autocommit 是开启状态，因此MySQL 在每个SQL语句之后执行一个提交，除非这个语句返回一个错误。如果一个语句返回一个错误，那么执行 commit 还是 rollback ，依赖这个具体的错误。

一个开启 auocommit 的会话（session）也可以通过START TRANSACTION / BEGIN  以及COMMIT / ROLLBACK 组合来执行一个多语句的事务。

如果一个会话通过 SET autocommit = 0 把 autocommit 模式关闭，这个会话总是一个事务打开。通过 COMMIT /  ROLLBACK 语句结束当前事务，同时开启一个新事务。

如果一个autocommit 模式关闭的会话在结束前没有明确提交最后的事务，那么MySQL 回滚这个事务。

部分SQL语句隐式结束一个事务，犹如你在执行SQL语句之前执行了COMMIT。

一个 COMMIT语句意味着在当前事务执行的改变会被持久化，并且对其他会话也变得可见。ROLLBACK 语句则取消当前事务的所有修改。COMMIT 与 ROLLBACK 释放当前事务设定的所有锁。

Grouping DML Operations with Transactions

默认，连接到MySQL server时，最初 autocommit 是开启状态，自动提交每个执行的SQL语句。如果你有使用其他数据库系统的经验，他们的标准操作是提交一系列的 DML 语句，然后一起提交或者回滚他们，因此可能感觉这种操作模式比较罕见。

为了使用多语句事务，通过 SQL 语句 SET autocommit = 0 把 autocommit 关闭，然后通过适当的 COMMIT / ROLLBACK 语句结束事务。当 autocommit 开启时，也可以通过 START TRANSACTION 开启事务，COMMIT / ROLLBACK 结束对应的事务。

Transactions in Client-Side Languanges

在 API 中，如 PHP/ Perl DBI/ JDBC/ ODBC 或者是MySQL 标准的C接口，你可以像其他SQL语句（如SELECT/INSERT）一样使用strings类型发送事务控制语句（如COMMIT）到 MySQL 服务端。部分API 还提供单独特定的事务提交、回滚函数及方法。

### c. 一致性无锁读

一致性读意味着InnoDB 使用多版本，使得查询是在数据库的一个时间点的快照上执行。查询能够看到在那个时间点之前提交的事务所产生的改变。那个时间之后改变的提交，或者改变未提交，这个查询都将认为未改变。这条规则的例外是，这个查询可以看到当前事务中之前的改变。这个例外导致如下反常的情况：如果你更新了一个表的某些行，SELECT 将看到对应更新行的最新版本，但是对于其他未更新的行，它可能看到的还是较老的版本。如果其他会话同时更新相同表，那么你可能看到的MySQL的状态是非常反常的，因为在数据库中从来没有过这种状态。

如果事务隔离级别是“可重复读（REPEATABLE READ）”，所有在相同事务中的一致性读都读取相同的快照，这个快照是在这个事务中的第一个读建立的。你可以通过提交当前的事务，然后再冲洗提交新的查询来得到针对你查询的最新的快照。

针对“读提交（READ_COMMITED）”隔离级别，在事务中的每个一致性读设定与读取自己新的快照。

在“读提交”与“可重复读”隔离级别InnoDB处理SELECT语句时，一致性读是默认模式。一致性读并不会给给它访问的表（table）设定任何锁。因此其他会话可以在同一时间自由修改这些表，即使这些表正在执行一致性读。

假设你在运行这默认的“可重复读”隔离级别。当你提交一个一致性读（也就是，一个普通的SELECT 语句），InnoDB 给你的事务一个时间点（timepoint），你看到数据库将依赖这个时间来决定。如果在你的时间点分派之后，其他的会话删除一个行并提交，你不会认为这行已删除。插入、更新也类似，处理方式相同。

你可以通过这些方式来提前这个时间点（timepoint），提交事务之后紧接着执行另外的 SELECT 或者 START TRANSACTION WITH CONSISTENT SNAPSHOT 来显式生成快照。

这称之为多版本并发控制（multi-versioned concurrency control）.

如果你想要看到数据库“最新”的状态，可以使用“读提交”（READ_COMMITED）隔离级别或者是加锁读（locking read）:

SELECT * FROM t FOR SHARE;

对于“读提交”（READ_COMMITED）隔离级别，在事务中的每个一致性读会设定并且读取自己的新快照。对于 LOCK IN SHARE MODE：SELECT 会阻塞（block）直到包含最新行的事务结束。

一致性读，在某些 DDL 语句上是不行的。

i. 一致性读在 DROP TABLES 上并不能工作，因为 MySQL 不能使用一个已经被删除的表，同时InnoDB 也会销毁这个表。
ii. 一致性读在 ALTER TABLE 上并不能工作，因为这时会创建一个原始表的临时表，当临时表构建完成后原来的表会被删除。当你在事务中重新提交一个一致性读时，在新表中的行则不可见，因为这些这个事务持有的快照中的行已经不存在了。这种情况下，这个事务会返回一个错误：ERR_TABLE_DEF_CHANGE, “表定义已经改变，请重试事务”。

在子句中，读的类型根据选择而不同，如 INSERT INTO ... SELECT, UPDATE ... (SELECT), 以及 CREATE TABLE ... SELECT ，这些语句并没有指定 FOR UPDATE 或者 LOCK IN SHARE MODE:

i. 默认，InnoDB 使用更强的锁，同时SELECT 部分将类似 READ_COMMITED 默认执行，也就是每次都是一致性读，甚至在相同的事务中设置及读取自己新的快照。
ii. 如果要在这些情况下，使用一致性读，打开 innodb_locks_unsafe_for_binlog选项，设置事务隔离级别为 READ_UNCOMMITED, READ_COMMITED, 或者 REPEATABLE_READ（只要不是 SERIALIZABLE 就可以）。这时，在读取到的这些行上则不会设置任何锁。

### d. 加锁读

如果你查询数据，然后在先相同的事务中执行 INSERT 或者 UPTATE 相关的数据，那么普通的 SELECT 语句则不能给予足够的保护。其他可以更新或者删除你刚刚查询的相同行。InnoDB 提供2种类型的加锁读，提供额外的安全性：

* SELECT ... LOCK IN SHARE MODE

    在读取的行上设定共享模式锁。其他事务可以读取这些行，但是不能修改他们直到你的事务提交。如果这些行中的任何行被其他事务改变，但是还没提交，你的查询将会等待直到那个事务结束，然后你的查询将使用最新的值。

* SELECT ... FOR UPDATE

    对于查询对应的索引记录，将对这些行及任何关联的索引条目（entries）加锁，这与你在这些行上执行一个 UPDATE 语句是相同的。那么对于更新这些行的事务、执行 SELECT ... LOCK IN SHARE MODE 事务、或者在某些事务隔离级别下的，这些都将会阻塞。一致性读则会忽略这些记录上的所有锁。（旧版本的记录不可能被加锁；通过对记录在内存中副本应用 undo logs来重构这些版本的较早版本的数据 ）

当处理树形结构（tree-structured）或者是图形结构（graph-structured）的数据时，无论是单表还是分布在多个表中，这些子句特别有用。

所有通过 LOCK IN SHARE MODE 与 FOR UPDATE 设定的锁，在这个事务提交或是回滚时被释放。

注意：只有当 autocommit 是关闭情况下，才可能有加锁的读（也就是要么通过 START TRANSACTION 开启事务，要么设置 autocommit 为 0）

加锁读的子句并会不会读嵌套的子查询对应的行加锁，除非子查询自身指定了加锁读。例如，下面的语句并不会对表 t2 加锁。

SELECT  * FROM t1 WHERE c1 = (SELECT c1 FROM t2) FOR UPDATE;

为了对表 t2 加锁，我们可以对子查询添加一个加锁读：

SELECT * FROM t1 WHERE c1 = (SELECT c1 FROM t2 FOR UPDATE) FOR UPDATE;

#### 加锁读示例

 假如你要在表 child 中插入一个新记录，同时确保这个记录在表 parent 表中有一个对应的记录。你的应用程序代码可以确保整个操作序列的引用完整性。

首先，使用一致性读查询表 parent 验证对应的parent 行是否存在。你能安全地在表 child 中安全的插入 child 行吗？不行的，因为其他会话可能在你 SELECT 与 INSERT 语句之前删除 parent 行，你完全不知道。

为了避免这种可能的问题，执行 LOCK IN SHARE MODE 的 SELECT:

SELECT * FROM parent WHERE NAME = 'Jones' LOCK IN SHARE MODE;

在 LOCK IN SAHRE MODE 的查询返回parent 记录 'Jones' 之后，你可以安全增加child记录到 CHILD 表，然后提交事务。任何试图在PARENT表对应记录上获取互斥锁的事务都将等待，直到你完成，也就是知道所有表的数据处于一个一致的状态。

比如另一个示例，考虑在CHILD_CODES 表中有一个整形的计数字段counter，用于给每个添加到表 CHILD 的的记录生成一个唯一的标识符。不要使用一致性读与共享模式的读去读这个计数字段之前的值，因为数据库的2个用户可能看到相同的counter值，这时如果2个事务试图使用相同的标识符添加行到 CHILD 表时将产生一个“duplicate-key”错误。

在这里，LOCK IN SHARE LOCK MODE 并不是一个好的解决方案，因为如果2个用户同时读取到这个 counter，那么至少其中一个当它试图去更新 couter时陷入死锁状态。

为了实现读、增加 counter，首先使用 FOR UPDATE 执行一个加锁读，然后增加 counter。如下：
SELECT counter_field FROM child_nodes FOR UPDATE;
UPDATE child_codes SET counter_filed = counter_field + 1;

SELECT ... FOR UPDATE 读取最新有效的数据，在读取到的行上设定互斥锁。因此，它与 UPDATE 语句在相同的记录上设定了相同的锁。

之前的描述仅仅是 SELECT ... FOR UPDATE 工作的一个例子。在MySQL 中，生成唯一标识符的特定任务实际上可以通过使用一个简单的访问来完成。

UPDATE child_codes SET counter_field = LAST_INSERT_ID(counter_field + 1);
SELECT LAST_INSERT_ID();

这个 SELECT 语句精简可以获取到这个标识符信息（特指当前连接）。它不能访问其他表。

## InnoDB 中不同SQL语句使用的锁

加锁读、UPDATE 、 DELETE 通常对在SQL语句处理过程中扫描的每个索引记录上设置记录锁。与在语句中是否有WHERE条件并没有太大的关系。InnoDB 并不会记住准确的WHERE 条件，而是只知道扫描的索引范围。锁通常是指 next-key 锁，即阻塞在相应记录之前的间隙插入新记录。然后，gap locking 可以被显式关闭，也将导致 next-key 锁不生效。事务的隔离级别也会影响使用哪种锁。

如果在一个查询中使用到了非聚簇索引并且设定了互斥性的索引记录锁，那么InnoDB也会查找其对应的聚簇索引记录，并且给他们加锁。

如果你的查询语句中没有合适的索引可以使用，MySQL 必须扫描整个表来处理这个这个查询，那么这个表中的每行都会加锁，进而阻塞其他用户针对这个表的所有插入操作。创建一个好的索引时非常重要的，以便让你的查询不需要扫描太多的行。

### InnoDB 设置的锁类型如下

* a. SELECT ... FROM 是一致性读，读取一个数据库的快照，并不会设定任何锁（除非事务隔离级别是 SERIALIZABLE）。对于 SERALIZABLE 级别，这个查询针对遇到的索引记录上设置共享的next-key锁。然后，对于使用唯一索引查找一个唯一的行，那么则使用 index record 锁。

* b. 对于 SELECT ... FOR UPDATE 或者是 SELECT ... LOCK IN SHARE MODE, 扫描的行加锁，同时对于结果集中不符合条件的行释放锁。（比如，他们不匹配WHERE子句的给定条件）。然后，有些情况下，行锁可能并不会立即释放，因为在查询执行期间结果集与原始数据关系丢失了。例如，在 UNION 中，从一个表中查询到的（同时加锁的）行，在评估是否满足结果集之前插入到一个临时表中。在这种情况下，临时表中中的行与原始表中的行之间的关系丢失了，因此后者的行的锁（原始表）在查询执行完成之前并不会被释放。

* c. SELECT ... LOCK IN SHARE MODE 在查询遇到的所有索引记录上设置共享的next-key锁。然而，如果为了查询一个唯一行使用到唯一索引时使用 index record 锁。

* d. SELECT ... FOR UPDATE 在查询遇到的所有索引记录上设置互斥next-key锁。然后，如果为了查询一个唯一行使用到唯一索引时使用 index record 锁。

    对于查询遇到的索引记录，SELECT ... FOR UPDATE 则阻塞 其他会话执行 SELECT ... LOCK IN SHARE MODE 以及在某些事务隔离级别下的读。一致性读会忽略读试图记录上的任何锁。

* e. UPDATE .... WHERE ... 在查询遇到的所有记录上设置互斥的next-key锁。然而，如果使用到唯一索引时，则为 index record 锁。

* f. 当 UPDATE 修改了一个聚簇索引记录，那么在影响的非聚簇索引记录上也会有隐式的锁。当在插入新的非聚簇索引记录之前执多key检查，以及当插入新的非聚簇索引记录时，UPDATE 操作在对应的非聚簇索引记录上添加共享锁。

* g. DELETE FROM ... WHERE ... 在查询遇到的所有索引记录上设置互斥next-key 锁。然而，如果使用到唯一索引时，则为 index record 锁。

* h. INSERT 在插入的行上设置互斥锁。这个锁是 index record 锁，并不是 next-key锁（即，并没有 gap 锁），同时不会阻止其他会话在插入行之前的 gap 中插入新记录。

    在插入新记录之前，一种称为插入意图间隙锁（insert intention gap lock）被设置。这个锁标识一种插入的意图。如果多个事务插入到相同的索引间隙(index gap) ，但并不插入相同的位置时，他们彼此之前并不需要等待。假如，有2个索引记录，其值分别为 4、7。不同的事务试图分别插入值为5、6的记录时，在插入的行上获取互斥锁（exclusive lock）之前，每个事务都使用插入意图锁（insert intention lock）对4-7之间的间隙（gap）加锁，但是并不会彼此阻塞，因为这些行并不冲突。

    如果发生键冲突（duplicate-key error）错误，那么在这个记录上设置一个共享锁。当多个会话插入相同的行，但是有另外的会话已经获取到了互斥锁（exclusive lock），那么这个共享锁的使用会导致死锁（deadlock）。当另外的事务删除行时也会发生。假如，InnoDB 表 t1 结构如下：

    ```sql
        CRAETE TABLE t1(i INT, PRIMARY KEY (I)) ENGINE = InnoDB;
    ```

    现在假设三个会话一次执行下面的操作：

    Session 1:

    ```sql
        START TRANSACTION;
        INSERT INTO t1 VALUES(1);
    ```

    Session 2:

    ```sql
        START TRANSACTION;
        INSERT INTO t1 VALUES(1);
    ```

    Session 3:

    ```sql
        START TRANSACTION;
        INSERT INTO t1 VALUES(1);
    ```

    Session 1:

    ```sql
        ROLLBACK;
    ```

    Session 1 的第一个操作在对应的行上获取到了一个互斥锁（exclusive lock）。Session 2 与 Session 3 都得到一个键冲突（duplicate-key）错误，他们都在这个行上获取到一个共享锁（shared lock）。当 Session 1 回滚时，它将释放此记录上的互斥锁（exclusive lock），持有排队共享锁（shared lock）的 Session 2 与 Session 3 将被授权。在这个点，Session 2 与 Session 3 将进入死锁（deadlock）：二者都不能获取到互斥锁（exclusive lock），因为另一个会话持有共享锁（shared lock）。

    如果表中已经包含一条记录（值为：1）时，相同的情形也会发生。3个会话一次执行下面的操作：

    Sesssion 1:

    ```sql
        START TRANSACTION;
        DELETE FROM t1 WHERE i = 1;
    ```

    Session 2:

    ```sql
        START TRANSACTION;
        INSERT INTO t1 VALUES(1);
    ```

    Session 3:

    ```sql
        START TRANSACTION;
        INSERT INTO t1 VALUES(1);
    ```

    Session 1:

    ```sql
        COMMIT;
    ```

    Session 1 的第一步操作获取到了对应记录上的互斥锁（exclusive lock）。Session 2 与 Session 3 都产生一个键冲突（duplicate-key）错误，同时也都获取到对应记录上的共享锁。当 Session 1 提交时，它释放了此行上的互斥锁（exclusive lock），Session 2 与 Session 3 排队共享锁请求将被授予。此时，Session 2、Session 3 进入死锁（deadlock）状态：二者都不能获取到互斥锁（exclusive lock），因为另一个会话持有共享锁（shared lock）。

* i. INSERT .. ON DUPLICATE KEY UPDATE 不同于简单的 INSERT 。当一个键冲突发生时，在对应行上使用互斥锁而不是共享锁用户更新。对于多个主键值（primary key value），则使用互斥索引记录锁（exclusive index-record lock）。对于多个唯一键值（unique key value），则使用互斥 next-key 锁。

* j. 如果在一个唯一索引上，没有冲突时， REPLACE 执行行为与 INSERT 相同。否则，在对应记录上放置互斥的 next-key 锁用于更新。

* k. INSERT INTO T SELECT ... FROM S WHERE ... 在插入到 T 中每一行设置互斥索引记录锁并不包含间隙锁）。如果事务的隔离级别是 READ_COMMITTED ，或者 innodb_locks_unsafe_for_binlog 处于开启状态，同时事务的隔离隔离级别不是 SEARIALIZABLE，那么 InnoDb 在表 S 上使用一致性读（并没有锁）。否则，InnoDB 在表 S 的行上设置共享的 next-key 锁。在后面这种情况 InnoDB 必须加锁：在使用 statement-based binlog 滚动恢复时，每个SQL语句必须与之前一样准确的执行。

    CREATE TABLE ... SELECT ... 中的 SELECT 使用共享的 next-key 锁，或者是一致性读，如同 INSERT ... SELECT 一般。

    当 SELECT 用于这样的结构时，RELACE INTO t SELECT ... FROM s WHERE ... 或者是 UPDATE t ... WHERE col IN (SELECT ... FROM s ...)，InnoDB 在表 s 对应的上设置共享的 next-key 锁。

* l. 当初始化一个之前指定为 AUTO_INCREMENT 列时，InnoDB 会在关联了 AUTO_INCREMENT 列索引的末尾设定一个互斥锁（exclusive lock）。

    如果设定 innodb_autoinc_lock_mode = 0 ，当需要访问自增计数器（auto-increment counter）时，InnoDB 使用一个特殊的 AUTO-INC 表锁模式。在这种模式中，锁的获取与持有直到当前这条 SQL 语句的结束，而并不是整个事务的结束。当持有 AUTO-INC 表锁时，其他客户端不能插入到表中。innodb_autoinc_lock_mode = 1 时，批量插入也会发生相同的行为。innodb_autoinc_lock_mode = 2 时，表级 AUTO-INC 锁则不会被使用。

    InnoDB 获取之前已经初始化的 AUTO_INCREMENT 列的值时，并不设定任何锁。

* m. 如果在表中定义了一个外键约束（FOREGIN KEY constraint），任何要求检查约束条件的插入、更新、删除操作都会在相应记录上设定共享记录级别锁（shared record-level lock），用于检查约束。即使是约束失败的情况，InnoDB 也会设定这些锁。

* n. LOCK_TABLES 设定表级锁，但这个锁本身是属于在 InnoDB 层之上的 MySQL 层的锁，只是在 InnoDB 层调用设定而已。如果 innodb_table_locks=1(默认) 且 autocommit=0 时，InnoDB知道表锁，InnoDB 之上的 MySQL 层也知道行级锁（row-level locks）。

    另外，InnoDB 的自动死锁检测并不能检测到表锁中包含的死锁。同样，在这种情况下，MySQL 层并不了解行级锁，因此很有可能在其他会话当前获取到航迹锁的情况下，当前会话也可能获得表锁。然而，这并没有危及到事务的完整性。

* o. 如果 innodb_table_locks=1, 那么LOCK TABLES在每个表上获取的2个锁。除 MySQL 层的表锁之外，它也获取到了 InnoDB 的表锁。MySQL 4.1.2 之前的版本并不会获取到 InnoDB 的表锁；可以通过设定 innodb_table_locks=0 来选择旧的这种行为。如果不使用 InnoDB表锁，那么虽然表中的某些记录被其他的事务加锁，LOCK TABLES 依然可以完成。

    在 MySQL 5.7 中，innodb_table_lock=0 并不影响显式知道给你了表锁的语句 LOCK TABLES ... WRITE。但是对于隐式的 LOCK TABLES ... WRITE 或者是 LOCK TABLES ... READ 会有影响。

* p. 当事务提交或者是中止时，这个事务持有的所有的 InnoDB 锁会都会释放。因此当 autocommit=1时，在 InnoDB 表执行 LOCK_TABLES 并没有多大的意义，因为获取到的这个 InnoDB 表级锁将被立即释放。

* q. 你不能再事务的中对其他表加锁，因为 LOCK TABLES 执行一个隐式的 COMMiT 与 UNLOCK_TABLES 操作。

## 幻读（Phantom Rows）

幻读即在一个事务中，不同时间执行相同的查询产生不同的结果集。比如，如果一个 SELECT 执行 2 次，在第一次返回了一行，但是在第一次没有返回，这个行就是“幻读”行。

假如在 child 表上 id 列有一个索引，你想要读取，同时锁住大于100的所有行，目的是想在之后更新选择的行的某些字段值。

```sql
    SELECT * FROM child WHERE id > 100 FOR UPDATE;
```

这个查询从 id 大于 100 的第一个记录开始扫描索引。使这个表包含 id 等于 90 、102 的行。如果只在索引记录上设定锁，而不对间隙（这里指，90与102之前的间隙）中的插入操作加锁，那么另外一个会话就可以在这个表中插入 id 为 101 行。如果你将要在相同的事务中执行相同的 SELECT , 那么你将在这个查询的返回结果中看到一个 id 为 101 的新行（幻读）。
如果我们期望这些行作为一个数据条目，那么新的幻读行将违反了事务的隔离原则（在这个事务执行期间，已经读取的行不改变）。

为了阻止幻读，InnoDB 使用了一种称为 next-key 锁的算法，结合了记录锁（index-row locking）与间隙锁（gap locking）。InnoDB 以这样一种方式执行行级别锁，当它查找或者是扫描一个表的索引时，会在经过的索引记录上设定共享或者是互斥锁。因此，行级锁（row-level locks）实际上是索引记录锁（index-record locks）。此外，在索引记录（index-record）上的 next-key 锁也影响这个索引记录之前间隙的。也就是，next-key 锁是一个索引记录锁（index-record）加上一个索引记录之前的间隙锁（gap lock）。如果一个会话在一个索引的行 R 上拥有共享锁或者是互斥锁，那么其他会话则不能在索引排序的规则下在 R 之前立即插入新的索引记录。

当 InnoDB 扫描一个索引记录时，它也会对索引中最后一个记录之后的间隙加锁。在之前例子中，为了阻止任何大于 100 的记录插入到表中，InnoDB 设定的锁包括 id 值为 102 之后的间隙。

你可以在你的应用中使用 next-key 锁来实现唯一性检测：如果你使用共享模式读取数据，同时不想看到新插入的行，那么你可以安全的插入记录，在之前行读取过程中成功设置的 next-key 锁会阻止其他人同时插入与你相同的记录。因此，next-key 锁可以让你对表中一些不存在的记录加锁。

如之前“InnoDB 锁”章节讨论，间隙锁（Gap locking）也可以关闭。这将导致幻读问题，因为当间隙锁关闭后，其他会话可以在间隙中插入新行。

## InnoDB 死锁

死锁即不同的事务因为每个事务持有了对方需要的锁，而导致他们不能继续执行。因为这些事务都在等待对应资源变为可用，然而都不会释放其持有的锁。

当事务在多个表上加锁（通过诸如 UPDATE 或者是 SELECT ... FOR UPDATE 等这些语句），但是顺序不同时，那么死锁就可能发生。当这些语句在索引记录范围及间隙加锁，因为时序问题每个事务获取了其中一部分锁，但是未获取到其他锁，这是死锁可可能发生。

为了减少发生死锁的可能性，可以使用替代 LOCK TABLES 语句；保持插入、更新的数据足够小，以使他们不会长时间保持运行；当不同的事务更新多个表或者是大范围的行时，在每个事务中保持相同的操作顺序（比如，SELECT ... FOR UPDATE）；在使用 SELECT ... FOR UPDATE 与 UPDATE ... WHERE 语句的列上创建索引。事务的隔离级别并不会影响发生死锁的概率，因为隔离级别改变的是读取操作的行为，而死锁是由写操作引起。关于如何避免与从死锁条件（deadlock conditions）中恢复的更多信息，可以阅读“如何减少及处理死锁”一节。

当死锁检测开启（默认），死锁发生时，InnoDB 会检测条件（condition）进而回滚其中一个事务(受害者)。如果使用 innodb_deadlock_detect 配置选项关闭了死锁检测，InnoDB 依靠 innodb_lock_wait_timeout 设置来回滚处于死锁状态的事务。因此，甚至如果你的应用逻辑是正确的，你也必须要处理事务重试的情况。为了查看在InnoDB 中用户事务的最后一个死锁，可以使用 SHOW ENGINE INNODB STATUS 命令。如果事务结构或者是应用错误处理导致频繁的死锁，那么可以打开 innodb_print_all_deadlocks 配置项来打印关于所有死锁信息到 mysqld 错误日志中。关于死锁如何检测及处理的更多信息，可以参考“死锁检测及回滚”。

### 一个 InnoDB 死锁的例子

下面的例子描述当一个锁请求将产生死锁时，会产品一个错误。例子包含2个客户端，A 与 B。

首先，客户端 A 创建一个表，包含一行，然后开始一个事务。在事务中，A 通过共享模式的查找获取了一个 S 锁：

```sql

mysql> CREATE TABLE t (i INT) ENGINE = InnoDB;
Query OK, 0 rows affected (1.07 sec)

mysql> INSERT INTO t (i) VALUES(1);
Query OK, 1 row affected (0.09 sec)

mysql> START TRANSACTION;
Query OK, 0 rows affected (0.00 sec)

mysql> SELECT * FROM t WHERE i = 1 LOCK IN SHARE MODE;
+------+
| i    |
+------+
|    1 |
+------+

```

下一步，客户端 B 开始一个事务，试图删除表中的一行。

```sql

mysql> START TRANSACTION;
Query OK, 0 rows affected (0.00 sec)

mysql> DELETE FROM t WHERE i = 1;

```

删除操作获取一个 X 锁。这个锁不能被授权，因为它与客户端 A 持有的 S 锁是不兼容的，所以这个请求进入此行的锁请求队列，同时客户端 B 阻塞。

最后，客户端 A 又试图去执行删除操作：

```sql

mysql> DELETE FROM t WHERE i = 1;
ERROR 1213 (40001): Deadlock found when trying to get lock;
try restarting transaction

```

这里死锁的发生时因为客户端 A 需要一个X锁删除记录。然后，那个锁请求不能被授予，这是因为客户端 B 已经发送了一个 X 锁请求并且正在等待客户端 A 释放它的 S 锁。客户端 A 并不能把 S 锁更新到 X 锁，那是因为之前 B 已经请求了 X 锁。最终，InnoDB 针对其中一个客户端返回一个错误，同时释放它的锁。客户端返回这样的错误：

```sql

ERROR 1213 (40001): Deadlock found when trying to get lock;
try restarting transaction

```

在那是，另一个客户端的锁请求可以被成功授权，进而成功删除表中对应的行。

### 死锁检测及回滚

当死锁检测开启（默认值）时，InnoDB 自动检测事务死锁，同时回滚一个事务或者多个事务来打破死锁。InnoDB 试图找出一些小的事务进行回滚，事务的大小是有插入、更新、删除的行数决定的。

如果 innodb_table_locks = 1 (默认值)，且 autocommit = 0 时， InnoDB 也会关注表锁，其之上的 MySQL 层也知道行级锁（row-level locks）。否则，InnoDB 不能侦测到通过 MySQL LOCK TABLES 语句设定的表锁或者由 InnoDB 以外的存储引擎设置的锁。通过设置 innodb_lock_wait_timeout 系统变量的值来解决这些情况。

当 InnoDB 执行一个完整的事务回滚时，这个事务设置上所有锁都会被释放。然后，如果仅仅是其中一个SQL语句执行错误而回滚，那么这个语句设置的部分锁可能依然是保留的。这是因为 InnoDB 以一种特定的形式来存储行锁，它并不知道之后哪些语句设定了哪些锁。

如果在事务中，一个 SELECT 调用了一个存储的函数，函数内的某个语句执行失败时，整个外层语句都会回滚。此外，如果之后执行 ROLLBACK，那么整个事务则回滚。

如果在 InnoDB Monitor 输出中的 LATEST DETECTED DEADLOCK 部分包含这样开始的消息，“TOO DEEP OR LONG SEARCH IN THE LOCK TABLE WAITS-FOR GRAPH, WE WILL ROLL BACK FOLLOWING TRANSACTIOIN,” 这表明在等待列表（wait-for list）中的事务个数已经到达了 200 的限制。超过 200 个事务的等待列表会作为死锁来对待，同时试图检测等待列表的事务则回滚。如果加锁的线程必须查看超过100万个由在等待列表中事务加的锁时也会发生相同的错误。

对于组织数据库操作来避免死锁的技术，可以参考“InnoDB中死锁”章节。

#### 关闭死锁检测

在高并发的系统中，当大量的线程等待相同的锁时，死锁检测会使系统变慢。有时，关闭死锁检测，以及当死锁发生时依靠 innodb_lock_wait_timeout 设置来使事务回滚可能会更高效。死锁检测可以通过 innodb_deadlock_detect 配置选项来关闭。

### 如何减少及处理死锁

这个小节是基于关于在“死锁检测及回滚”章节中的一些概念来构建的。它解释了如何组织数据库操作来最小化死锁及应用程序中必须处理的之后的错误。

在事务数据库中，死锁（Deadlocks）是一个传统的问题，但是他们并不危险，除非他们非常频繁以至于你完全不能执行某些事务。通常，你必须这么处理你的引用程序，以便如果因为死锁而导致事务回滚时总是准备期重新提交事务。

InnoDB 使用自动的行级锁（row-level locking）。即使仅仅只有 insert 或者是 delete 单行，也可能产生死锁。那是因为这些操作并不是真正的“原子性”的；他们自动在插入或者删除的索引记录上加锁。

你可以使用下面的技术对付死锁，减少他们发生的可能：

* 在任何时候，可以执行 SHOW ENGINE INNODB STATUS 命令确定最近产生死锁的原因。它可以帮助你调优你的应用程序以避免死锁。

* 如果频繁的死锁警告产生了一些影响，可以通过开启 innodb_print_all_deadlocks 配置选项收集更多调试信息（debuggin infomation）。关于每个死锁的信息（并不仅仅是最后一个）都会记录在 MySQL 的 error log。当你调试完毕时，关闭此选项。

* 总是准备重新提交因为死锁而失败的事务。死锁并不危险。只需要重试即可。

* 保持事务更小及持续时间更短，以减少冲突的可能性。

* 在执行了相关的改变之后，立即提交事务，以使得减少冲突。特别是，有未提交事务时，不要让交互式的 mysql 会话长时间打开。

* 如果你使用加锁读（SELECT ... FOR UPDATE 或者 SELECT ... LOCK IN SHARE MODE），尝试使用更低级别的事务隔离级别（诸如，READ_COMMITED）。

* 当在一个事务中修改多个表，或者是同一个表中不同范围的行时，每次使用一致的顺序这些操作。这样这些事务形成一个明确的队列，并不会产生死锁。例如，在你的应用程序中，把这些数据库操作封装为函数或者是存储的协程，而不是在不同的地方编写多个相似的 INSERT, UPDATE, DELETE 语句序列。

* 给你的表添加选择性更好的（well-chosen）索引。那么查询就会扫描更少的所有记录，因此也就设定更少的锁。使用 EXPLAIN SELECT 来确定 MySQL server 使用哪个索引以使得更适合你的查询。

* 使用更少的锁。如果你允许 SELECT 在旧的快照中返回数据，那么就不要给其添加子句 FOR UPDATE 或 LOCK IN SHARE MORE。在此处使用 READ COMMITED 隔离级别也是不错的选择，因为在相同的事务中每个一致性读都读取最新的快照。

* 如果以上这些都没有帮助，那么使用表级锁序列化你的事务。针对事务表（如 InnoDB 表）使用 LOCK TABLES 的正确方式是通过 SET autocommit = 0 开始一个事务（而不是 START TRANSACTION），接着 LOCK TABLES，直到你明确提交事务之前不要调用 UNLOCK TABLES。例如，如果你需要写入表 t1，同时从表 t2 读取数据，你可以这样做：

    ```sql

    SET autocommit=0;
    LOCK TABLES t1 WRITE, t2 READ, ...;
    ... do something with tables t1 and t2 here ...
    COMMIT;
    UNLOCK TABLES;

    ```

    表级锁阻止了并发更新表，从而避免死锁，但是对于繁忙的系统，响应能力会降低。

* 另外一个序列化事务的方式是创建一个仅仅包含单行的辅助信号（semaphore）表。使得每个事务在访问其他表之前更新这个表。以这种方式，所有的方式都以一种序列化的形式执行。注意，在这种情况下，InnoDB 死锁检测算法依然在工作，因为序列化的锁是一个行级锁。对于 MySQL 表级锁，必须用超时的方法来解决死锁。
