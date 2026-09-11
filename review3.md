## modules/apps/desktop/editor/src/App.vue:47 ( )
Явный глинт. Слишком большой смотри сколько у него методов он выводит и лог какой-то и нотис и все что угодно мне кажется имеет смысл разбить этот здоровенный компост на более мелкие и название у них тоже очень странные что они про что делают что значит дуинг кажется таски что значит going? Это опять куча герундиев которые остались Давай здесь нужно его разбить. Композебл и переименовать функции. 

## modules/apps/desktop/editor/src/agent-tab/kind.ts:22 (+)
Ты это называешь conversation так используй этот термин везде почему тебе какой-то аск ? почему у тебя аргумент функции называется Talk тип у него conversation иди унифицировай название переменных чтобы они могли если почему мы будем там делать conversation conversation?

## modules/apps/desktop/editor/src/agent-tab/kind.ts:24 ( )
Что такое Asked это ввод пользователя или что это?  Целый комментарий сверху нужно написать, чтобы объяснить что это такое. Давай нормально название переменной 

## modules/apps/desktop/editor/src/agent-tab/kind.ts:38 ( )
что такое place для толка? И что такое Talk вообще Это называется сессиям либо чат, либо еще что-то. Channel, Fred, и никак не place давай тоже переименовывай это 

## modules/apps/desktop/editor/src/agent-tab/kind.ts:47 ( )
landed чего? И что это за мапа streng of strengy? Вынесь это в тип хотя бы здесь, что здесь landed чего в чего у тебя даже три строчки комментариев нужно было добавить, чтобы объяснить что это такое. 

## modules/apps/desktop/editor/src/agent-tab/kind.ts:48 ( )
Что такое аскинг опять Герундий аскинг чего и почему это сет с тренингов ?

## modules/apps/desktop/editor/src/agent-tab/kind.ts:51 ( )
В комментарии написано каждый адрес, адрес чего, что такое аск что такое спрашивать что мы? Зачем какие-то адреса передаем, это вроде агентской табы, почему здесь адреса находятся. ?

## modules/apps/desktop/editor/src/agent-tab/places.ts (file-level)
что значит places я так понял ты здесь с линками работаешь значит это линки, а не places. Значит отсылки на что-то. Линки, но никак не Places, мы нигде не используем слово Places, Оно у тебя в Глоссарии вообще есть? Оно называется Линк. Тем более у тебя внутри объявлен интерфейс Link Target 

## modules/apps/desktop/editor/src/agent-tab/title.ts:4 (+)
но здесь ты наверно не первую линию берешь или что? У тебя же чуть сложнее логика чем первую линию взять ты там еще обрезаешь по моему по длине символов назови функцию нормально не знаю типа Shorton или еще что-нибудь. Ну у тебя мост есть даже функция Аргумент есть. Ты во первых здесь первую линию берешь, а ограничиваешь по длине это не просто первая линия это значит shorten что-нибудь либо exerbed подумай лучше 

## modules/apps/desktop/editor/src/agent-tab/types.ts:8 (+)
здесь название функциях опять либо наречие либо герундии давай исправляет 

## modules/apps/desktop/editor/src/agent-tab/types.ts:15 (+)
У тебя почти похожий тип уже есть в в Places файле определен зачем тебе два одинаковых типа зачем? У тебя в кайнде есть ссылка и у тебя здесь есть ссылка Зачем тебе два типа одинаковых? 

## modules/apps/desktop/editor/src/book-tab/BookTab.vue:74 (+)
Нужно похоже добавить еще одну секцию Hooks в файлах компонентов поскольку это явно не хелперы, здесь хуки нужно вверх вынести логику вынести в функции хелтеры сделает и здесь и в других местах тоже нужно переделать  

## modules/apps/desktop/editor/src/book-tab/BookTab.vue:96 (+)
так почему вот это вообще в отдельный компонент не вытащите? То есть весь бук контент что это такое мне кажется вообще отдельный компонент можно вытащить. Все что в transition в отдельный компонент тащить можно. 

## modules/apps/desktop/editor/src/book-tab/BookTab.vue:113 (+)
Что такое ADD? На текущей странице ну так и назови ее текущей страницей. 

## modules/apps/desktop/editor/src/book-tab/BookTab.vue:115 (+)
Что значит elseware? Это вообще про что? Кажется какой-то довольно глупый аргумент либо найди ему нормальное название либо смершего с чем-то другим. Он вообще про что? 

## modules/apps/desktop/editor/src/book-tab/markup.ts:17 (+)
Название очень странное что значит pointed ad что входит в markup что такое entry, почему он с тренга возвращается. Что значит point itted это ссылка куда-то Так и назови Get что-то это глагол должен быть а не наречие. 

## modules/apps/desktop/editor/src/book-tab/open.ts:32 (+)
Так у нас есть тип спам а зачем-то еще один какой-то богспан вводишь зачем тебе отдельные типы для этих вещей Который просто фром конвертирует в ту. Этот тип излишен, Зачем он здесь нужен? 

## modules/apps/desktop/editor/src/book-tab/open.ts:51 (+)
Что такое ED текущая страница или что? Ed это что? В чем оно измеряется оно что показывает? ?

## modules/apps/desktop/editor/src/book-tab/open.ts:52 (+)
Син чего? Что видено видено? 

## modules/apps/desktop/editor/src/book-tab/open.ts:54 (+)
Что значит стендинг? Это про что ? нормальное название дай нормальные перемены 

## modules/apps/desktop/editor/src/book-tab/open.ts:56 (+)
что такое elseware он про что это подсветка или что И зачем здесь букспанк? Когда у нас уже конкретный спанк для этого есть. У нас уже есть типспан зачем мы новый переизобретаем 

## modules/apps/desktop/editor/src/book-tab/open.ts:58 (+)
это что возвращает номер страницы Если номер страницы, он так и назови что значит Page? Page что? 

## modules/apps/desktop/editor/src/book-tab/open.ts:69 (+)
Reading что? Что? Ридинг Что значит чтение? Чтение чего? Зачем здесь опять ненужный тип? Нормально назови функцию точнее перемену нормально назови 

## modules/apps/desktop/editor/src/book-tab/open.ts:127 (+)
что значит речь Что это опять за странный глагол? Он о чем говорит? И опять же здесь у тебя спан используется на каких-то целей вводил букв спам зачем нам букспан если ты здесь используешь 

## modules/apps/desktop/editor/src/book-tab/open.ts:146 (+)
судя по количеству экспортов здесь это нужно разбивать на более мелкие кажется и более мелкие кампазы у тебя уже здесь слишком много кампазов, чтобы Сколько их здесь 15 считаешь нормальным ?

## modules/apps/desktop/editor/src/book-tab/pagination.ts:14 (+)
ЭД это что? Это байты, это страницы. В чем? В чем это измеряется? ? функция должна называться глаголом 

## modules/apps/desktop/editor/src/book-tab/pagination.ts:24 (+)
Named что? Зачем ты опять наречие здесь используешь Глагол должен быть 

## modules/apps/desktop/editor/src/book-tab/pagination.ts:30 (+)
функция должна называться глаголом . 

## modules/apps/desktop/editor/src/book-tab/types.ts:32 (+)
зачем нам отдельный испанка когда у нас уже есть тип спан для чего мы его вводим ?

## modules/apps/desktop/editor/src/book-tab/types.ts:35 (+)
зачем мы какие-то печатные страницы где мы это используем если это просто номер страницы почему от них не хранить просто как страница почему Printted? Что значит Printted? Это EPAB? Там нет ничего напечатанного почему этот поле так называется? 

## modules/apps/desktop/editor/src/book-tab/types.ts:36 (+)
Если это количество страниц в книге, то так и назови 

## modules/apps/desktop/editor/src/book-tab/types.ts:37 (+)
Что значит PageBites? Это размер одной страницы в байтах или размер всех страниц в байтах. 

## modules/apps/desktop/editor/src/book-tab/types.ts:40 (+)
Зачем вам какую-то обратную совместимость сделать и что такое ADD? Add что? Почему это везде Number, а тут строка? Кажется лишний какой-то лишнее поле 

## modules/apps/desktop/editor/src/book-tab/types.ts:50 (+)
Что значит READN Markup Ты конкретно получаешь результат в какой-то строке поэтому это должен быть гетто не Рид и что значит СИН опять? Видно если это какой-то тип ну назови его Либо переименуй 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:12 (+)
что значит EDD это номер страницы или что ?

## modules/apps/desktop/editor/src/document-tab/highlights.ts:13 (+)
Что за трабл у нас уже есть договоренность мы называем ошибкой новое слово зачем ты выдумаешь?  Разве не прочь это сделать? Функция которая возвращает? Зачем нам реф? Через трабл передавать? Должен быть хендлер который composable дергает кажется так более чисто. Как это ниже сделал. 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:29 (+)
что это что эта функция делает почему называется Highlight Она что-то производит? Что она делает? 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:37 (+)
это не название функции по ней вообще не понятно что она делает переименуй 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:53 (+)
Это что здесь такое? Если это про подсветку, другую типа не текущий Current так сделай у подсветки просто разные варианты разный флаг там добавим зачем ты целое поле для этого выделил 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:55 (+)
что значит олсу это вообще про что 

## modules/apps/desktop/editor/src/document-tab/highlights.ts:57 (+)
Что значит речь это не глагол назвали нормально дай 

## modules/apps/desktop/editor/src/document-tab/navigation.ts (file-level)
Почему в файле объявлен Composable? А имя файла не соответствует названию того, что это Composable. Все Composable, которые начинаются с Use должны лежать в файлах. Которые также начинаются с Use, чтобы вы могли понять, что это Composable. 

## modules/apps/desktop/editor/src/document-tab/navigation.ts:8 (+)
Что значит Add страница или что это номер страницы ? переименуй давай 

## modules/apps/desktop/editor/src/document-tab/navigation.ts:10 (+)
go что о книге если книги то открыть страницу или что Потому что Go! Go ?куда? 

## modules/apps/desktop/editor/src/document-tab/navigation.ts:15 (+)
Next что? Страница следующая или что это может быть глагол который понятно объясняет что мы делаем 

## modules/apps/desktop/editor/src/document-tab/navigation.ts:16 (+)
Здесь тоже самое back что назад что страница или что ?

## modules/apps/desktop/editor/src/document-tab/open.ts (file-level)
почему название файла не соответствует имени Composable Файл должен соответствовать имени Composable объявленного в нем. 

## modules/apps/desktop/editor/src/document-tab/open.ts:36 (+)
Мне кажется несмотря на то что зарефакторили много чего вытащили из этого Композа была можно что-то еще там посмотри, выритерни их все равно штук десять и опять же почему название файла не соответствует названию Composable 

## modules/apps/desktop/editor/src/document-tab/types.ts:21 (+)
зачем нам два типа ? зачем нам здесь page highlight? Мы что в типе не можем сделать rect типа highlights опциональными зачем нам еще один тип вводить?  Если это прям необходимо методами получения данных поскольку они разные то еще ok но в целом как-то как будто бы не нужен 

## modules/apps/desktop/editor/src/document-tab/types.ts:27 (+)
Зачем здесь обратная совместимость? Хайлайт page page hylight это жесть просто удали это никакой обратной совместимости ты что делаешь 

## modules/apps/desktop/editor/src/document-tab/types.ts:35 (+)
Зачем нам типы для обратной совместимости? Мы что, У нас нет никаких релизов, какую обратную совместимость ты делаешь? Нафига нам этот мусор здесь? 

## modules/apps/desktop/editor/src/document-tab/viewport.ts:13 (+)
это что? Это про что? Про новые страницы или про что? ?

## modules/apps/desktop/editor/src/document-tab/viewport.ts:14 (+)
Это вообще про что ? Син что? Переименуй непонятно он про что вообще 

## modules/apps/desktop/editor/src/document-tab/viewport.ts:29 (+)
зачем эта уродская конструкция? У тебя уже есть функция, которая названа правильно 

## modules/apps/desktop/editor/src/document-tab/viewport.ts:36 (+)
это не глагол.

## modules/apps/desktop/editor/src/document-tab/viewport.ts:37 (+)
эта штука URL похоже возвращает ну так и назови мы везде это eMoch называем нахрена ты еще

## modules/apps/desktop/editor/src/files-tab/FilesTab.vue:34 ( )
Что значит renaiming? Если это bulliev значит оно должно начинаться с is 

## modules/apps/desktop/editor/src/files-tab/FilesTab.vue:142 ( )
это не трабл, а error

## modules/apps/desktop/editor/src/files-tab/listing.ts:27 ( )
такое ощущение что введение этого типа нового нафиг не нужно теперь это через Entry не можем вывести ?

## modules/apps/desktop/editor/src/files-tab/listing.ts:42 ( )
Что значит landed in нормальное название функции, уж нихрена не понятно о чем наговорить 

## modules/apps/desktop/editor/src/files-tab/listing.ts:46 ( )
Это не глагол это опять наречие. 

## modules/apps/desktop/editor/src/files-tab/listing.ts:55 ( )
Это не глагол это опять наречие 

## modules/apps/desktop/editor/src/files-tab/listing.ts:66 (+)
почему имя файла не названо в соответствии с именем Composable. Мне кажется Композа был делает слишком много ты посмотри сколько у него экспортов посмотреть сколько этот компост был возвращает через return такое ощущение что пора разбивать 

## modules/apps/desktop/editor/src/files-tab/listing.ts:74 ( )
это называется error

## modules/apps/desktop/editor/src/files-tab/open.ts:28 (+)
судя по количеству ретернов и количество типов которые ты возвращаешь это нужно разбить слишком здоровенный composable и он опять не назван по имени файла 

## modules/apps/desktop/editor/src/files-tab/open.ts:166 (+)
зачем все это экспорте через алиасы? и ивенты у нас начинаются со слова ON Handler у нас начинается с on потом уже пример такой onHandler

## modules/apps/desktop/editor/src/files-tab/types.ts:14 (+)
для этого типа у нас уже есть Для этого поля уже есть тип Точка называется. 

## modules/apps/desktop/editor/src/files-tab/types.ts:18 (+)
нам действительно нужен вот такой тип Intoybefore точно так вот нам нужно это сделать или нет способа лучше сделать ?

## modules/apps/desktop/editor/src/files-tab/types.ts:21 (+)
похоже нужно разбить на более мелкие и использовать как несколько зависимостей и переименую функции чтобы было понятно что они делают функции мы называем глаголами 

## modules/apps/desktop/editor/src/files-tab/types.ts:22 (+)
Что значит лендс? 

## modules/apps/desktop/editor/src/files-tab/types.ts:32 (+)
Что значит seis переименуй, чтобы было понятно что он делает 

## modules/apps/desktop/editor/src/files-tab/types.ts:40 (+)
а ты уверен что это нормально иметь такой здоровенный тип? Почему бы не разбить его на более мелкие ? проверено именно вот этого типа кажется все функции нужно переименовать на них нихрена не понятно о чем говорят что значит что значит Кенран что значит over в общем то надо разбить и попереименовать все. Жесткий артефакт нужен. 

## modules/apps/desktop/editor/src/files-tab/types.ts:65 (+)
Что значит спрашивать? Это о чем вообще? Переименуй должен быть глагол 

## modules/apps/desktop/editor/src/flashcards-deck-tab/DeckTab.vue (file-level)
такое ощущение что отсюда можно много чего зарефакторить файл получился очень длинным и много чего в нем есть мне кажется можно зарефакторить уже 

## modules/apps/desktop/editor/src/flashcards-deck-tab/DeckTab.vue:23 ( )
если это ошибка то назови это ошибкой 

## modules/apps/desktop/editor/src/flashcards-deck-tab/DeckTab.vue:29 ( )
Что значит offred? Offred что назови нормально поле 

## modules/apps/desktop/editor/src/flashcards-deck-tab/DeckTab.vue:123 ( )
знаете что такое похоже тоже можно вытащить в отдельный компонент 

## modules/apps/desktop/editor/src/flashcards-deck-tab/DeckTab.vue:129 ( )
это явно можно вытащить в отдельный компонент что он здесь торчит 

## modules/apps/desktop/editor/src/flashcards-deck-tab/answers.ts:125 ( )
так но название функции явно нужно переименовать через Герундий глаголы должны быть глаголы и всякие наречия герундий из названия функций должно быть понятно что они делают 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.actions.ts:19 (+)
это похоже Composable соответственно используя Use переименую и функцию имя файла 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.actions.ts:25 (+)
Проверь все функции здесь, чтобы не было дублирующихся тоже здесь куча функций которые похожи друг друга дублируют

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.actions.ts:26 (+)
название опять неправильное зачем тебе в Present Simple мы глаголами это называем что значит Ads? Это глагол должен быть из которого понятно что он делает 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.actions.ts:45 (+)
зачем здесь две одинаковые функции одна из них одна из них Remove Section а вторая Removes Section зачем две функции которые делают одно и то же? 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.schedule.ts:12 (+)
почему файл не назван в часть композа была ?переименуй файл везде так используй 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.schedule.ts:57 (+)
название функции здесь сломано нет нигде нормальных глаголов которые было понятно не делают какие-то наречия в очередной раз переименую чтобы все было понятно чего они возвращают 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.ts:34 (+)
так судя по тому сколько там всего возвращается такое ощущение что фиги передает сюда и это нужно разбить на более мелкие компосты было. Если это composable на lzave файл соответственно с именем composable 

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.ts:180 ( )
что значит ментинг Что это значит минтинг чего? ?

## modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.ts:260 ( )
Из названия этих переменных вообще непонятно чего они возвращают переименуй их все 

## modules/apps/desktop/editor/src/flashcards-deck-tab/drawn.ts (file-level)
название файла вообще не говорит о том что здесь происходит 

## modules/apps/desktop/editor/src/flashcards-deck-tab/mutations.ts (file-level)
название файла вообще не чем не говорит что значит Motations ? Mutations чего? ? Такое ощущение что нужно разбить ею на более мелкие компазумбы зачем-то хранить все в одном файле у тебя куча разных действий в одном файле и миллион импортов теперь поэтому давай разбей это на более мелкие 

## modules/apps/desktop/editor/src/flashcards-deck-tab/mutations.ts:25 (+)
если минт это сгенерировать а1 так и назови айдидженератор или как то еще 

## modules/apps/desktop/editor/src/flashcards-deck-tab/reader.ts:14 ( )
Название типа тупое опять глупые наречия переименую в нормально название существительное функции глагола 

## modules/apps/desktop/editor/src/flashcards-deck-tab/reader.ts:74 ( )
Название функции тупое оно меняет внутреннее состояние, но ничего не возвращает соответственно какой-то сет 

## modules/apps/desktop/editor/src/flashcards-deck-tab/serialize.ts (file-level)
это явно нужно разбить на более мелкие композблы или еще что то вообще не ясно что это происходит куча разных функций, которые просто какие-то JSON возвращают. Или объекты 

## modules/apps/desktop/editor/src/flashcards-deck-tab/types.ts (file-level)
Зачем в одном файле определять столько типов? У тебя импортов куча идей и помойка получается давай разбивать все на мелкие вещи один файл один тип везде так действует 

## modules/apps/desktop/editor/src/flashcards-deck-tab/types.ts:69 (+)
Ты посмотри сколько здесь типов сколько здесь полей штук 30? Это явно нужно разбивать на более мелкие стейки Пускай стейк может состоять из нескольких.

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:31 ( )
и какой смысл если ты ну какой в этом смысл ты же можешь напрямую обращаться в шаблоне зачем тебе это, Для чего? Просто чтобы реактивность здесь потерять? 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:37 ( )
Sate Instad название нормально дай значит said inswer instad дай нормальное название а то у тебя просто какой-то конвертер одного в другое какой-то очень сложный конверт бесполезный сделать проще 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:81 ( )
это явно можно разбить нам кучу мелких компонентов 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:85 ( )
но это выглядит как один компонент который ты там два раза используешь вынеси это в отдельный компонент 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:99 ( )
Это выглядит как компоненты где то мы уже подобное видели. Имеет смысл тоже вынести 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:107 ( )
вложено их дело зачем? Почему один в другой нельзя сделать? Вынеси это в какой-то компонент layout 

## modules/apps/desktop/editor/src/flashcards-preset-tab/PresetTab.vue:118 ( )
но здесь явно какой-то костыли на костылях ты просто строчку переписываешь 

## modules/apps/desktop/editor/src/flashcards-preset-tab/core.ts (file-level)
у тебя куча функций которые делают разные почему-то в одном файле лежит Core явно это можно вынести в разные компазум 

## modules/apps/desktop/editor/src/flashcards-preset-tab/core.ts:17 ( )
но это тоже выглядит странно почему бы там в протоколе не объявить его нужным образом что здесь здесь приходится переписывать 

## modules/apps/desktop/editor/src/flashcards-preset-tab/core.ts:33 (+)
это тоже какой-то явный признак плохого дизайна нужно рефакторить весь этот файл 

## modules/apps/desktop/editor/src/flashcards-preset-tab/curve-slider/CurveSlider.vue (file-level)
Можно разбивать за целых 600 строчек миллион импортов полностью рефакторить 

## modules/apps/desktop/editor/src/flashcards-preset-tab/curve.ts (file-level)
Почему у нас есть два файла? Один называется curve второй называется curves как мы должны понимать что куда записывать это явно эти два файла нужно зарефакторить полностью 

## modules/apps/desktop/editor/src/flashcards-preset-tab/flight.ts (file-level)
это выглядит как базовый функционал всего редактора какая-то запись непонятно что она здесь делает вообще для чего это нужно и выглядит как что-то очень переусложненное Если какая-то логика это нужно вынести отсюда. Вообще выглядит как какая-то дичь какой-то большой адский костыль 

## modules/apps/desktop/editor/src/flashcards-preset-tab/kind.ts:97 ( )
название функции опять Наречие давай глаголы здесь сделай 

## modules/apps/desktop/editor/src/flashcards-preset-tab/open.ts (file-level)
в файле опять типичная помойка куча типов разных. И composable который возвращает все 

## modules/apps/desktop/editor/src/flashcards-preset-tab/open.ts:20 (+)
это не проблема это называется Error у нас даже нарратив для этого есть 

## modules/apps/desktop/editor/src/flashcards-preset-tab/open.ts:192 (+)
тебе опять Composable возвращает миллион штук нужно его явно разбить на более мелкие 

## modules/apps/desktop/editor/src/flashcards-preset-tab/preset-settings/PresetSettings.vue (file-level)
этот файл нужно рефакторить здесь 300 с лишним строк разбивай на мелкие компоненты 

## modules/apps/desktop/editor/src/flashcards-preset-tab/types.ts (file-level)
опять в одном файле миллион типов нужно артефакты разбивать это на мелкие файлы 

## modules/apps/desktop/editor/src/flashcards-preset-tab/types.ts:126 (+)
какой-то определенной логики, где мы читаем файл почему здесь какая-то определенная своя логика определенные свои типы для чтения. Что за мусор здесь ?

## modules/apps/desktop/editor/src/flashcards-preset-tab/types.ts:134 (+)
то есть целая какая-то специальная логика записи файлов у тебя какая-то целая специальная логика записи файлов и вот для конкретно этого типа нужно исправлять это логика общего назначения зачем она здесь нужна конкретно здесь 

## modules/apps/desktop/editor/src/flashcards-preset-tab/types.ts:215 (+)
это адская жесть сколько тут 20 полей? Явно нужно разбивать на более мелкие надо раскидывать по файлы и названия полей, переименовывать все это дело. Функция это глаголы 

## modules/apps/desktop/editor/src/flashcards-stencil-tab/StencilTab.vue:89 ( )
где-то это уже видел похоже топоры выносить в отдельный компонент 

## modules/apps/desktop/editor/src/flashcards-stencil-tab/StencilTab.vue:126 ( )
для этого мы используем слово error. мне кажется мы этот стиль уже повторяем везде тысячу раз иди проверь к асопаровому артефакты этих компонентов 

## modules/apps/desktop/editor/src/flashcards-stencil-tab/stencilTabs.fields.ts:66 (+)
вот это вот здесь зачем? Что здесь происходит? ? Мы здесь перекладываем одно в другое и делаем какую-то обертку точно лучше способа нет ?

## modules/apps/desktop/editor/src/flashcards-stencil-tab/types.ts:11 (+)
вот этот стейк нужно разбивать сколько здесь 30 полей нужно разбивать на мелкие стейты  названия полей там явно какие-то хендлеры есть хендлеры мы делаем через onHandler название полей дублируется что там за кип Keep Mind точнее Keep mind, В чем разница? Take time file, Sharts Close куча дублирующих полей, add face и add's face Move Field и Move Field огромное количество полей которые друг друга дублируют полностью рефакторить весь тип 

## modules/apps/desktop/editor/src/media-url-tab/embed/Embed.vue (file-level)
файл почти 300 строчек рефакторить и разбивать на более мелкие более мелкие компоненты 

## modules/apps/desktop/editor/src/note-tab/conflict.ts (file-level)
так что у нас конфликты только в Not tab появляются ? нужно эти артефакты выносить куда-нибудь в Shert 

## modules/apps/desktop/editor/src/note-tab/conflict.ts:20 (+)
что значит saying of название функции должно быть глаголом это дичь какая-то 

## modules/apps/desktop/editor/src/note-tab/kind.ts:166 ( )
название функций не глаголы. они должны быть глаголами понятно что они делают начнем здесь две функции Колос потом Cold ?

## modules/apps/desktop/editor/src/note-tab/kind.ts:182 ( )
зачем этот комментарий здесь вообще Он что дает? Бешеная дичь какая-то здесь написано ?

## modules/apps/desktop/editor/src/note-tab/maker.ts (file-level)
опять набор функций непонятно для чего в композе было бы сделали или еще что-то что за мусор здесь 

## modules/apps/desktop/editor/src/note-tab/noteTypes.ts (file-level)
опять набор типов без набор типов в одном файле раскидай по файлам назови 

## modules/apps/desktop/editor/src/note-tab/notes.ts:155 ( )
смотри сколько ретёрнов у этого композа было Сколько? Двадцать тридцать? Давай рефакторы на более мелкие 

## modules/apps/desktop/editor/src/note-tab/queue.ts (file-level)
почему сохранение заметок здесь какую-то отдельную логику имеет у нас уже было с пресетами такая же фигня и нот Q вообще ни о чем не говорит очередь чего на что на рефакторе и выносить в шеррит мод ядро,  Рефакторить и выносить в ядро делать унифицированным. 

## modules/apps/desktop/editor/src/plex-tab/PlexTab.vue:128 ( )
Что значит дроп нейм? Что это такое? По полю вообще непонятно 

## modules/apps/desktop/editor/src/plex-tab/PlexTab.vue:129 ( )
Почему пальцы передаются отдельно? Почему в этом ноде как типам не передать? Ну то есть есть есть нода у нее есть тип и у этого типа есть парты 

## modules/apps/desktop/editor/src/plex-tab/kind.ts:111 ( )
название функции вообще опять не глаголы и опять какие-то наречия 

## modules/apps/desktop/editor/src/plex-tab/open.ts:21 (+)
а компас был возвращает 30 30. Полей рефакторит разбивать на мелкие название файла должно соответственно название Composable разбить на мелкие 

## modules/apps/desktop/editor/src/plex-tab/parts.ts:12 (+)
что такое плекс парт ? переименуй и название файла соответствует названию композулу 

## modules/apps/desktop/editor/src/plex-tab/tickets.ts (file-level)
Тикет что такое ticket? в рамках Плекс это вообще нигде не объявлено нету такого термина откуда ты его взял? ?Переименую в нормальное дай нормальное название если это Composable то это Composable то есть пользуй нормальную конвенцию для composable имя файла соответствующее дай 

## modules/apps/desktop/editor/src/plex-tab/types.ts (file-level)
опять файл с миллионом типов раздели на мелкие один тип один файл. 

## modules/apps/desktop/editor/src/plex-tab/types.ts:30 (+)
Здесь опять огромное количество зависимостей нужно разделить на более мелкие типы 

## modules/apps/desktop/editor/src/plex-tab/types.ts:48 (+)
Стей тоже разбух Разбить на более мелкие. 

## modules/apps/desktop/editor/src/settings-file-tab/SettingsFileTab.vue:44 ( )
Так выглядит как очередной компонент который можно пере использовать Вы уже видели его раз десять 

## modules/apps/desktop/editor/src/settings-file-tab/SettingsFileTab.vue:48 (+)
это явно можно вынести в отдельный компонент кажется мы уже видели. Зачем нам два вот сверху есть компоненты здесь еще один что между ними разница 

## modules/apps/desktop/editor/src/settings-tab/SettingsTab.vue (file-level)
файл на 600 строк разбивай на более мелкие как минимум по секциям 

## modules/apps/desktop/editor/src/settings-tab/kind.ts (file-level)
в файле огромное количество типов выноси типы по файлам один файл один тип 

## modules/apps/desktop/editor/src/settings-tab/kind.ts:17 ( )
название типа вообще ни о чем не говорит это настройки или что? Здесь порядка 30 полей разбивай на более мелкие 

## modules/apps/desktop/editor/src/settings-tab/setting-row/SettingRow.vue:16 ( )
название поля вообще ни чем не говорит. если у нас уже есть имя и детали того зачем это поле ?

## modules/apps/desktop/editor/src/shared/answers.ts (file-level)
Файл этот про что он делает? Конвертирует ошибку в строку непонятная дичь какая-то вообще > Непонятно что делает и название файла тоже ни о чем не говорит 

## modules/apps/desktop/editor/src/shared/artifacts.ts (file-level)
миллион типов в одном файле, раскидать нужно разбить 

## modules/apps/desktop/editor/src/shared/command/address.ts:19 (+)
этот альяс зачем? Удали алиас используй нормальное название 

## modules/apps/desktop/editor/src/shared/command/handlers.ts:19 ( )
зачем этот здоровенный объект? Его нужно разделить на более мелкие. Это капец тут все потом будет собрано и импортов будет миллион Иди исправляй. 

## modules/apps/desktop/editor/src/shared/core.ts:27 ( )
GOD OBJECT! нужно разбить его на более мелкие знаете почему здесь в нем все хранится он хранит вообще все разбит на более мелкие по файлам один файл один кусок. 

## modules/apps/desktop/editor/src/shared/file.ts (file-level)
Если это только к файловому менеджеру относится то лежит в Shared 

## modules/apps/desktop/editor/src/shared/file.ts:45 (+)
зачем этот бессмысленный алиас используя название типа 

## modules/apps/desktop/editor/src/shared/icons.ts (file-level)
этот файл вообще зачем ? зачем все иконки тащить в один файл Это что дает? ? Почему иконки не использовать по месту? 

## modules/apps/desktop/editor/src/shared/note.ts (file-level)
Если это про заметки то зачем это здесь? Что здесь общего? Зачем здесь соседи считаются? Спан здесь объявлен, объект из домена, опять он объявлен  Зачем алиасы для типов делать? 

## modules/apps/desktop/editor/src/shared/vaults.ts (file-level)
Опять куча типов в одном файле раскидать разбить на более мелкие опять зачем-то алиасы тут есть 

## modules/apps/desktop/editor/src/shared/vaults.ts:43 (+)
зачем этот Алиас Почему правильно не использовать сразу? 

## modules/apps/desktop/editor/src/shared/words.ts (file-level)
Нафига это здесь Зачем мы в каких-то местах используем Шеред? В каких то местах прям в компоненте потом будем делать нормальную i18n l10n Кажется это мусором каким-то либо раскидаю по компонентам либо все в Шеред перетащи  лучше по компонентам раскидать 
